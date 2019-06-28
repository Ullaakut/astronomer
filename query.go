package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenk/backoff"

	"github.com/ullaakut/disgo"
)

const (
	year                   = 24 * time.Hour * 365
	rateLimitSleepDuration = time.Hour / 5000
)

type context struct {
	repoOwner          string
	repoName           string
	githubToken        string
	cacheDirectoryPath string

	// Number of years from now to scan for contributions.
	// Less years means significantly less requests, but
	// ignores value from users who were active since
	// the beginning of GitHub.
	scanUntilYear int
}

func fetchStargazers(ctx context) ([]user, error) {
	var (
		requestBody    string
		cursor         string
		users          []user
		rateLimitSleep time.Duration
	)

	// Inject constant values into request body.
	requestBody = strings.Replace(fetchStargazersRequest, "$repoOwner", ctx.repoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.repoName, 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	client := &http.Client{}

	defer disgo.EndStep()

	// Iterate on lists of users.
	var page int
	for {
		disgo.StartStepf("Fetching stargazers %d to %d", page*usersPerRequest, (page+1)*usersPerRequest)
		page++
		// Inject variables into request body.
		paginatedRequestBody := strings.Replace(requestBody, "$pagination", fmt.Sprint(usersPerRequest), 1)

		// If this isn't the first request, inject the cursor value.
		if cursor != "" {
			paginatedRequestBody = strings.Replace(
				paginatedRequestBody,
				fmt.Sprintf("stargazers(first:%d){", usersPerRequest),
				fmt.Sprintf("stargazers(first:%d,after:\\\"%s\\\"){", usersPerRequest, cursor),
				1)
		}

		// Get all user contributions since 2008.
		currentYear := time.Now().Year()
		for i := 0; currentYear-i > ctx.scanUntilYear-1; i++ {
			from := time.Date(currentYear-i, time.January, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(year - 1*time.Second)

			yearlyRequestBody := strings.Replace(paginatedRequestBody, "$dateFrom", from.Format(iso8601Format), 1)
			yearlyRequestBody = strings.Replace(yearlyRequestBody, "$dateTo", to.Format(iso8601Format), 1)

			req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer([]byte(yearlyRequestBody)))
			if err != nil {
				return nil, disgo.FailStepf("unable to prepare request: %v", err)
			}

			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.githubToken))
			req.Header.Set("User-Agent", "Astronomer")

			resp, err := getCache(ctx, req, pagination(page, currentYear-i))
			if err != nil {
				return nil, disgo.FailStepf("unable to get cached file: %v", err)
			}

			response, responseBody, _ := parseResponse(resp)

			// If the request was not found in the cache, try to fetch it until it works
			// or until the limit of 20 attempts is reached.
			if resp == nil {
				var attempts int
				backoff.Retry(func() error {
					// If we reached 20 attempts, give up.
					attempts++
					if attempts >= 20 {
						disgo.Errorln("Failed to fetch users from GitHub API too many times.")
						return nil
					}

					// If rate limit was reached, sleep before making a request.
					// If rate limit was not reached, rateLimitSleep will be set to zero.
					time.Sleep(rateLimitSleep)

					resp, err = client.Do(req)
					if err != nil {
						return fmt.Errorf("unable to fetch stargazers: %v", err)
					}

					if resp == nil {
						return errors.New("nil response")
					}

					response, responseBody, err = parseResponse(resp)
					if err != nil {
						return err
					}

					if len(response.Errors) != 0 || response.ErrorMessage != "" {
						if response.ErrorMessage != "" {
							return errors.New(response.ErrorMessage)
						}
						return errors.New(response.Errors[0].Message)
					}

					if len(response.Repository.Stargazers.Users) == 0 {
						resp.Body.Close()
						disgo.Debugln("Reached end of user list")
						return nil
					}

					return nil
				}, backoff.NewConstantBackOff(15*time.Second))
			}

			if response == nil {
				return nil, fmt.Errorf("failed to fetch users. last body recieved: %s", responseBody)
			}

			// We reached the end.
			if len(response.Repository.Stargazers.Users) == 0 {
				resp.Body.Close()
				disgo.Debugln("Reached end of user list")
				return users, nil
			}

			users = updateUsers(users, *response, currentYear-i)

			cursor = response.Repository.Stargazers.Meta.cursor()

			if len(response.Errors) != 0 || response.ErrorMessage != "" {
				disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
				return nil, disgo.FailStepf("failed to fetch users. last body recieved: %s", responseBody)
			}

			err = putCache(ctx, req, pagination(page, currentYear-i), responseBody)
			if err != nil {
				return users, disgo.FailStepf("unable to write user data to cache: %v", err)
			}

			if response.RateLimit.Remaining <= 10 {
				disgo.Debugln("Rate limit reached, slowing down requests")
				rateLimitSleep = rateLimitSleepDuration
			}
		}

		if cursor == "" {
			disgo.Debugln("Reached end of user list")
			break
		}

		disgo.EndStep()
	}

	return users, nil
}

func pagination(page, year int) string {
	return fmt.Sprintf("-%d-%d", page, year)
}

func parseResponse(resp *http.Response) (*listStargazersResponse, []byte, error) {
	if resp == nil {
		return nil, nil, errors.New("unable to parse nil response")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("unable to read response body: %v", err)
	}

	var response listStargazersResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		disgo.Errorf("Response: %s\n", responseBody)
		resp.Body.Close()
		return nil, nil, fmt.Errorf("unable to unmarshal stargazers: %v", err)
	}

	if len(response.Errors) != 0 {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("error while querying user data: %v [%s:%s]", response.Errors[0].Message, response.Errors[0].Extensions.argumentName, response.Errors[0].Extensions.name)
	}

	return &response, responseBody, nil
}

func updateUsers(users []user, response listStargazersResponse, year int) []user {
	var (
		found    bool
		newUsers []user
	)

	newUsers = response.Repository.Stargazers.Users

	// Update users if they already exist in the list.
	for idx := range users {
		for _, u := range newUsers {
			if users[idx].Login == u.Login {
				users[idx].yearlyContributions[year] = u.Contributions.ContributionCalendar.TotalContributions + u.Contributions.PrivateContributions

				users[idx].Contributions.PrivateContributions += u.Contributions.PrivateContributions
				users[idx].Contributions.TotalCommitContributions += u.Contributions.TotalCommitContributions
				users[idx].Contributions.TotalIssueContributions += u.Contributions.TotalIssueContributions
				users[idx].Contributions.TotalPullRequestContributions += u.Contributions.TotalPullRequestContributions
				users[idx].Contributions.TotalPullRequestReviewContributions += u.Contributions.TotalPullRequestReviewContributions
				users[idx].Contributions.TotalRepositoryContributions += u.Contributions.TotalRepositoryContributions

				found = true
			}
		}
	}

	if !found {
		for idx := range newUsers {
			newUsers[idx].yearlyContributions = make(map[int]int)
			newUsers[idx].yearlyContributions[year] = newUsers[idx].Contributions.ContributionCalendar.TotalContributions + newUsers[idx].Contributions.PrivateContributions
		}

		users = append(users, newUsers...)
	}

	return users
}

// func loadState(ctx context) ([]users, error) {

// }

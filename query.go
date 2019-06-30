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

const year = 24 * time.Hour * 365

var (
	rateLimitSleepDuration time.Duration

	// blacklistedUsers contains the list of users that can't be
	// fetched from the GitHub API. When one of these users is found
	// in a list request, he must be skipped when fetching user contributions
	// or astronomer will be stuck due to constant API timeouts.
	blacklistedUsers = []string{
		"jstrachan",
	}
)

// context represents the
type context struct {
	repoOwner          string
	repoName           string
	githubToken        string
	cacheDirectoryPath string

	// If details is set to true, astronomer will print
	// additional factors such as percentiles.
	details bool
}

// fetchStargazers fetches the list of cursors to iterate upon to
// fetch stargazer contributions.
func fetchStargazers(ctx context) ([]string, error) {
	var (
		stargazers     []stargazers
		lastCursor     string
		page           int
		rateLimitSleep time.Duration
	)

	// Inject constant values into request body.
	requestBody := strings.Replace(fetchUsersRequest, "$repoOwner", ctx.repoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.repoName, 1)
	requestBody = strings.Replace(requestBody, "$pagination", fmt.Sprint(listPagination), 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	client := &http.Client{}

	defer disgo.EndStep()

	for {
		page++

		var response *listStargazersResponse

		paginatedRequestBody := requestBody
		if lastCursor != "" {
			paginatedRequestBody = strings.Replace(
				paginatedRequestBody,
				fmt.Sprintf("stargazers(first:%d){", listPagination),
				fmt.Sprintf("stargazers(first:%d,after:\\\"%s\\\"){", listPagination, lastCursor),
				1)
		}

		req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer([]byte(paginatedRequestBody)))
		if err != nil {
			return nil, disgo.FailStepf("unable to prepare request: %v", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.githubToken))
		req.Header.Set("User-Agent", "Astronomer")

		resp, err := getCache(ctx, req, listFilePagination(page))
		if err != nil {
			return nil, disgo.FailStepf("unable to get cached file: %v", err)
		}

		response, responseBody, _ := parseResponse(resp)

		// If the request was not found in the cache, try to fetch it until it works
		// or until the limit of 20 attempts is reached.
		if resp == nil {
			disgo.StartStepf("Listing stargazers %d to %d", (page-1)*listPagination, (page)*listPagination)

			var attempts int
			backoff.Retry(func() error {
				// If we reached 20 attempts, give up.
				attempts++
				if attempts >= 20 {
					disgo.Errorln("Failed to fetch stargazers from GitHub API too many times.")
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
					return nil
				}

				return nil
			}, backoff.NewConstantBackOff(15*time.Second))
		}

		if response == nil {
			return nil, fmt.Errorf("failed to fetch stargazers. last body recieved: %s", responseBody)
		}

		stargazers = append(stargazers, response.Repository.Stargazers)

		lastCursor = response.Repository.Stargazers.Meta.cursor()

		if len(response.Errors) != 0 || response.ErrorMessage != "" {
			disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
			return nil, disgo.FailStepf("failed to fetch user contributions. last body recieved: %s", responseBody)
		}

		err = putCache(ctx, req, listFilePagination(page), responseBody)
		if err != nil {
			return nil, disgo.FailStepf("unable to write user contribution data to cache: %v", err)
		}

		// Set the rate limit sleep duration depending on the token's limit.
		rateLimitSleepDuration = time.Hour / time.Duration(response.RateLimit.Limit)

		if response.RateLimit.Remaining <= 10 {
			disgo.Debugln("Rate limit reached, slowing down requests")
			rateLimitSleep = rateLimitSleepDuration
		}
	}

	cursors := getCursors(stargazers)

	return cursors, nil
}

// Return the appropriate cursors to be used by the fetchContributions function
// according to the value of ${contribPagination}. Also makes sure not to include
// any page of users containing blacklisted individuals.
func getCursors(sg []stargazers) []string {
	var (
		skip       bool
		totalUsers int
		cursors    []string
	)

	for _, stargazers := range sg {
		var currentPageUsers int
		for _, user := range stargazers.Users {
			if isBlacklisted(user.Login) {
				skip = true
			}
		}

		// Iterate through list of stargazers, and add a cursor for every
		// ${contribPagination} users, unless one of the users within the current
		// page is blacklisted, in which case we skip the whole page.
		if totalUsers > 0 && totalUsers%contribPagination == 0 {
			if !skip {
				cursors = append(cursors, stargazers.Meta[currentPageUsers].Cursor)
			} else {
				skip = false
			}
		}

		currentPageUsers++
		totalUsers++
	}

	return cursors
}

func isBlacklisted(user string) bool {
	for _, blacklistedUser := range blacklistedUsers {
		if user == blacklistedUser {
			return true
		}
	}

	return false
}

// fetchContributions fetches the contribution data of a list of stargazers.
// ctx contains the scanned context of the astronomer command.
// untilYear is the year until which to scan for contribuitons.
func fetchContributions(ctx context, cursors []string, untilYear int) ([]user, error) {
	var (
		page           int
		requestBody    string
		users          []user
		rateLimitSleep time.Duration
	)

	// Inject constant values into request body.
	requestBody = strings.Replace(fetchContributionsRequest, "$repoOwner", ctx.repoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.repoName, 1)
	requestBody = strings.Replace(requestBody, "$pagination", fmt.Sprint(contribPagination), 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	client := &http.Client{}

	defer disgo.EndStep()

	// Iterate on lists of users.
	for {
		page++

		// If this isn't the first request, inject the cursor value.
		paginatedRequestBody := requestBody
		if page > 0 {
			paginatedRequestBody = strings.Replace(
				paginatedRequestBody,
				fmt.Sprintf("stargazers(first:%d){", contribPagination),
				fmt.Sprintf("stargazers(first:%d,after:\\\"%s\\\"){", contribPagination, cursors[page]),
				1)
		}

		// Get all user contributions for each year.
		currentYear := time.Now().Year()
		for i := 0; currentYear-i > untilYear-1; i++ {
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

			resp, err := getCache(ctx, req, contribFilePagination(page, currentYear-i))
			if err != nil {
				return nil, disgo.FailStepf("unable to get cached file: %v", err)
			}

			response, responseBody, _ := parseResponse(resp)

			// If the request was not found in the cache, try to fetch it until it works
			// or until the limit of 20 attempts is reached.
			if resp == nil {
				disgo.StartStepf("Fetching user contributions from users %d to %d for year %d", (page-1)*contribPagination, (page)*contribPagination, currentYear-i)

				var attempts int
				backoff.Retry(func() error {
					// If we reached 20 attempts, give up.
					attempts++
					if attempts >= 20 {
						disgo.Errorln("Failed to fetch user contributions from GitHub API too many times.")
						return nil
					}

					// If rate limit was reached, sleep before making a request.
					// If rate limit was not reached, rateLimitSleep will be set to zero.
					time.Sleep(rateLimitSleep)

					resp, err = client.Do(req)
					if err != nil {
						return fmt.Errorf("unable to fetch stargazer contributions: %v", err)
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
						return nil
					}

					return nil
				}, backoff.NewConstantBackOff(15*time.Second))
			}

			if response == nil {
				return nil, fmt.Errorf("failed to fetch user contributions. last body recieved: %s", responseBody)
			}

			users = updateUsers(users, *response, currentYear-i)

			if len(response.Errors) != 0 || response.ErrorMessage != "" {
				disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
				return nil, disgo.FailStepf("failed to fetch user contributions. last body recieved: %s", responseBody)
			}

			err = putCache(ctx, req, contribFilePagination(page, currentYear-i), responseBody)
			if err != nil {
				return users, disgo.FailStepf("unable to write user contribution data to cache: %v", err)
			}

			if response.RateLimit.Remaining <= 10 {
				disgo.Debugln("Rate limit reached, slowing down requests")
				rateLimitSleep = rateLimitSleepDuration
			}
		}

		if page == len(cursors) {
			disgo.Debugln("Reached end of user list")
			break
		}

		disgo.EndStep()
	}

	return users, nil
}

func contribFilePagination(page, year int) string {
	return fmt.Sprintf("-%d-%d", page, year)
}
func listFilePagination(page int) string {
	return fmt.Sprintf("-list-%d", page)
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
		return nil, responseBody, fmt.Errorf("unable to unmarshal stargazers: %v", err)
	}

	if len(response.Errors) != 0 {
		resp.Body.Close()
		return nil, responseBody, fmt.Errorf("error while querying user data: %v [%s:%s]", response.Errors[0].Message, response.Errors[0].Extensions.argumentName, response.Errors[0].Extensions.name)
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

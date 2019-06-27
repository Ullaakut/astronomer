package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ullaakut/disgo"
)

const year = 24 * time.Hour * 365

type context struct {
	repoOwner          string
	repoName           string
	githubToken        string
	cacheDirectoryPath string
}

func fetchStargazers(ctx context) ([]user, error) {
	var (
		requestBody string
		cursor      string
		users       []user
	)

	// Inject constant values into request body.
	requestBody = strings.Replace(fetchStargazersRequest, "$repoOwner", ctx.repoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.repoName, 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	client := &http.Client{}

	// Iterate on lists of users.
	for {
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
		for i := 0; currentYear-i > 2007; i++ {
			from := time.Date(currentYear-i, time.January, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(year - 1*time.Second)

			yearlyRequestBody := strings.Replace(paginatedRequestBody, "$dateFrom", from.Format(iso8601Format), 1)
			yearlyRequestBody = strings.Replace(yearlyRequestBody, "$dateTo", to.Format(iso8601Format), 1)

			req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer([]byte(yearlyRequestBody)))
			if err != nil {
				return nil, fmt.Errorf("unable to prepare request: %v", err)
			}

			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.githubToken))
			req.Header.Set("User-Agent", "Astronomer")

			// resp, err := getCache(ctx, req)
			// if err != nil {
			// 	return nil, fmt.Errorf("unable to get cached file: %v", err)
			// }

			// if resp == nil {
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("unable to fetch stargazers: %v", err)
			}
			// }

			defer resp.Body.Close()

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("unable to read response body: %v", err)
			}

			var response listStargazersResponse
			err = json.Unmarshal(responseBody, &response)
			if err != nil {
				disgo.Errorf("Response: %s\n", responseBody)
				return nil, fmt.Errorf("unable to unmarshal stargazers: %v", err)
			}

			if len(response.Errors) != 0 {
				return nil, fmt.Errorf("error while querying user data: %v [%s:%s]", response.Errors[0].Message, response.Errors[0].Extensions.argumentName, response.Errors[0].Extensions.name)
			}

			// We reached the end.
			if len(response.Repository.Stargazers.Users) == 0 {
				return users, nil
			}

			users = updateUsers(users, response, currentYear-i)

			cursor = response.Repository.Stargazers.Meta.cursor()

			err = putCache(ctx, req, resp)
			if err != nil {
				return users, fmt.Errorf("unable to write user data to cache: %v", err)
			}
		}

		if cursor == "" {
			break
		}
	}

	return users, nil
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

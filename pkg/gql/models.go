package gql

import (
	"time"

	"github.com/Ullaakut/disgo"
)

const (
	// Request 100 users per query when listing stargazers.
	listPagination = 100

	// Request 20 users per query when fetching contribution data.
	contribPagination = 20

	// ISO8601 time format used by the GitHub API.
	iso8601Format = "2006-01-02T15:04:05Z"

	// Request to list users. Low cost in terms of rate limiting.
	fetchUsersRequest = `{"query" : "{
			rateLimit {
				limit
				remaining
			}
			repository(owner: \"$repoOwner\", name: \"$repoName\") {
				stargazers(first: $pagination) {
					edges {
						cursor
					}
					nodes {
						login
					}
				}
			}
		}"}`

	// Request to fetch user contributions. Expensive in terms of rate limiting.
	// Fetching more than 20 users at a time is pretty much a guaranteed timeout.
	fetchContributionsRequest = `{"query" : "{
			rateLimit {
				remaining
			}
			repository(owner: \"$repoOwner\", name: \"$repoName\") {
				stargazers(first: $pagination) {
					edges {
						cursor
					}
					nodes {
						login
						createdAt
						contributionsCollection(from: \"$dateFrom\", to: \"$dateTo\") {
							restrictedContributionsCount
							totalIssueContributions
							totalCommitContributions
							totalRepositoryContributions
							totalPullRequestContributions
							totalPullRequestReviewContributions
							contributionCalendar {
								totalContributions
							}
						}
					}
				}
			}
		}"}`
)

// User represents a github user who starred a repository. It is
// public because this model is the output of the Fetch methods of
// this package.
type User struct {
	Login         string        `json:"login"`
	CreatedAt     string        `json:"createdAt"`
	Contributions contributions `json:"contributionsCollection"`

	YearlyContributions map[int]int
}

// DaysOld returns the amount of days since this user created their
// GitHub account.
func (u User) DaysOld() float64 {
	creationDate, err := time.Parse(iso8601Format, u.CreatedAt)
	if err != nil {
		disgo.Errorln("Unexpected date time format from GraphQL API:", err)
	}

	return time.Since(creationDate).Hours() / 24
}

type listStargazersResponse struct {
	response `json:"data"`

	ErrorMessage string     `json:"message"`
	Errors       []gqlError `json:"errors"`
}

type gqlError struct {
	Extensions gqlErrorExtension `json:"extensions"`
	Message    string            `json:"message"`
}

type gqlErrorExtension struct {
	Name         string `json:"name"`
	ArgumentName string `json:"argumentName"`
}

type response struct {
	Repository repository `json:"repository"`
	RateLimit  rateLimit  `json:"ratelimit"`
}

type rateLimit struct {
	Limit     int    `json:"limit"`
	Cost      int    `json:"cost"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"resetAt"`
}

type repository struct {
	Stargazers stargazers `json:"stargazers"`
}

type stargazers struct {
	Users []User   `json:"nodes"`
	Meta  metaData `json:"edges"`
}

type metaData []meta

// cursor returns the last cursor, or none if the metadata is empty.
// The cursor is used for pagination purposes, it represents the ID
// of the last fetched node, and lets the API know that we want to
// offset our query by the position of that node.
func (m metaData) cursor() string {
	if len(m) == 0 {
		return ""
	}

	return m[len(m)-1].Cursor
}

type meta struct {
	Cursor string `json:"cursor"`
}

type contributions struct {
	PrivateContributions                int `json:"restrictedContributionsCount"`
	TotalIssueContributions             int `json:"totalIssueContributions"`
	TotalCommitContributions            int `json:"totalCommitContributions"`
	TotalRepositoryContributions        int `json:"totalRepositoryContributions"`
	TotalPullRequestContributions       int `json:"totalPullRequestContributions"`
	TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`

	ContributionCalendar contributionCalendar `json:"contributionCalendar"`
}

type contributionCalendar struct {
	TotalContributions int `json:"totalContributions"`
}

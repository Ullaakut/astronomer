package main

import (
	"time"

	"github.com/ullaakut/disgo"
)

const (
	usersPerRequest        = 20
	iso8601Format          = "2006-01-02T15:04:05Z"
	fetchStargazersRequest = `{"query" : "{
			rateLimit {
				limit
				cost
				remaining
				resetAt
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

type repositoryStarScan struct {
	users     []user
	updatedAt time.Time
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
	code         string `json:"code"`
	name         string `json:"name"`
	typeName     string `json:"typeName"`
	argumentName string `json:"argumentName"`
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
	Users []user   `json:"nodes"`
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

type user struct {
	Login         string        `json:"login"`
	CreatedAt     string        `json:"createdAt"`
	Contributions contributions `json:"contributionsCollection"`

	yearlyContributions map[int]int
}

func (u user) daysOld() float64 {
	creationDate, err := time.Parse(iso8601Format, u.CreatedAt)
	if err != nil {
		disgo.Errorln("Unexpected date time format from GraphQL API:", err)
	}

	return time.Since(creationDate).Hours() / 24
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

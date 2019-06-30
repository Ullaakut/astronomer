package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestBody(t *testing.T) {
	tests := map[string]struct {
		baseRequest string
		repoOwner   string
		repoName    string
		pagination  int

		expectedBody string
	}{
		"fetch users request": {
			baseRequest: fetchUsersRequest,
			repoOwner:   "ullaakut",
			repoName:    "cameradar",
			pagination:  42,

			expectedBody: `{"query":"{ rateLimit{ limit remaining } repository(owner:\"ullaakut\",name:\"cameradar\"){ stargazers(first:42){ edges{ cursor } nodes{ login } } } }"}`,
		},
		"fetch contributions request": {
			baseRequest: fetchContributionsRequest,
			repoOwner:   "ullaakut",
			repoName:    "camerattack",
			pagination:  84,

			expectedBody: `{"query":"{ rateLimit{ remaining } repository(owner:\"ullaakut\",name:\"camerattack\"){ stargazers(first:84){ edges{ cursor } nodes{ login createdAt contributionsCollection(from:\"$dateFrom\",to:\"$dateTo\"){ restrictedContributionsCount totalIssueContributions totalCommitContributions totalRepositoryContributions totalPullRequestContributions totalPullRequestReviewContributions contributionCalendar{ totalContributions } } } } } }"}`,
		},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			ctx := context{
				repoOwner: test.repoOwner,
				repoName:  test.repoName,
			}

			requestBody := buildRequestBody(ctx, test.baseRequest, test.pagination)

			assert.Equal(t, test.expectedBody, requestBody)
		})
	}
}

func TestGetCursors(t *testing.T) {
	sg := stargazers{
		Users: []user{{Login: "titi"}, {Login: "toto"}, {Login: "tete"}, {Login: "tata"}, {Login: "tutu"}},
		Meta:  metaData{{Cursor: "titi"}, {Cursor: "toto"}, {Cursor: "tete"}, {Cursor: "tata"}, {Cursor: "tutu"}},
	}

	blacklistedStargazers := stargazers{
		Users: []user{{Login: "jstrachan"}, {Login: "toto"}, {Login: "tete"}, {Login: "tata"}, {Login: "tutu"}},
		Meta:  metaData{{Cursor: "jstrachan"}, {Cursor: "toto"}, {Cursor: "tete"}, {Cursor: "tata"}, {Cursor: "tutu"}},
	}

	tests := map[string]struct {
		stargazers []stargazers
		totalUsers int

		expectedCursors []string
	}{
		"less users than pagination": {
			stargazers: []stargazers{
				sg,
			},
			totalUsers: 5,

			expectedCursors: nil,
		},
		"exactly as many users as pagination": {
			stargazers: []stargazers{
				sg, sg, sg, sg,
			},
			totalUsers: 20,

			expectedCursors: nil,
		},
		"more users than pagination": {
			stargazers: []stargazers{
				sg, sg, sg, sg,
				sg, sg, sg,
			},
			totalUsers: 35,

			expectedCursors: []string{"tutu"},
		},
		"way more users than pagination": {
			stargazers: []stargazers{
				sg, sg, sg, sg, sg, sg, sg, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
			},
			totalUsers: 160,

			expectedCursors: []string{"tutu", "tutu", "tutu", "tutu", "tutu", "tutu", "tutu"},
		},
		"blacklisted stargazers should dcause page skips": {
			stargazers: []stargazers{
				sg, sg, sg, sg, sg, sg, blacklistedStargazers, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
				sg, sg, sg, sg, sg, sg, sg, sg,
			},
			totalUsers: 160,

			expectedCursors: []string{"tutu", "tutu", "tutu", "tutu", "tutu", "tutu"},
		},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			cursors := getCursors(test.stargazers, test.totalUsers)

			assert.Equal(t, test.expectedCursors, cursors)
		})
	}
}

func TestUpdateUsers(t *testing.T) {
	tests := map[string]struct {
		users    []user
		response listStargazersResponse
		year     int

		expectedUsers []user
	}{
		"nil users": {
			users: nil,
			response: listStargazersResponse{
				response: response{
					Repository: repository{
						Stargazers: stargazers{
							Users: []user{
								{Login: "titi", Contributions: contributions{
									PrivateContributions: 84,
									ContributionCalendar: contributionCalendar{
										TotalContributions: 42,
									},
								}},
								{Login: "toto", Contributions: contributions{
									PrivateContributions: 21,
									ContributionCalendar: contributionCalendar{
										TotalContributions: 67,
									},
								}},
							},
						},
					},
				},
			},
			year: 2019,

			expectedUsers: []user{
				{Login: "titi", Contributions: contributions{
					PrivateContributions: 84,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 42,
					},
				}, yearlyContributions: map[int]int{
					2019: 126,
				}},
				{Login: "toto", Contributions: contributions{
					PrivateContributions: 21,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 67,
					},
				}, yearlyContributions: map[int]int{
					2019: 88,
				}},
			},
		},
		"update already existing users": {
			users: []user{
				{Login: "titi", Contributions: contributions{
					PrivateContributions: 84,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 42,
					},
				}, yearlyContributions: map[int]int{
					2019: 126,
				}},
				{Login: "toto", Contributions: contributions{
					PrivateContributions: 21,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 67,
					},
				}, yearlyContributions: map[int]int{
					2019: 88,
				}},
			},
			response: listStargazersResponse{
				response: response{
					Repository: repository{
						Stargazers: stargazers{
							Users: []user{
								{Login: "titi", Contributions: contributions{
									PrivateContributions: 84,
									ContributionCalendar: contributionCalendar{
										TotalContributions: 42,
									},
								}},
								{Login: "toto", Contributions: contributions{
									PrivateContributions: 21,
									ContributionCalendar: contributionCalendar{
										TotalContributions: 67,
									},
								}},
							},
						},
					},
				},
			},
			year: 2018,

			expectedUsers: []user{
				{Login: "titi", Contributions: contributions{
					PrivateContributions: 168,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 42,
					},
				}, yearlyContributions: map[int]int{
					2019: 126,
					2018: 126,
				}},
				{Login: "toto", Contributions: contributions{
					PrivateContributions: 42,
					ContributionCalendar: contributionCalendar{
						TotalContributions: 67,
					},
				}, yearlyContributions: map[int]int{
					2019: 88,
					2018: 88,
				}},
			},
		},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			users := updateUsers(test.users, test.response, test.year)

			assert.Equal(t, test.expectedUsers, users)
		})
	}
}

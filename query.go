package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cenk/backoff"
	"github.com/ullaakut/disgo"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
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

	// If fastMode is set to true and that the repository
	// has more than ${fastModeStars} stargazers, pick
	// ${fastModeStars}/${contribPagination} number of cursors
	// randomly from all cursors.
	fastMode bool

	// Amount of stars to scan in fastMode. Will be used only if
	// fastMode is enabled.
	stars uint

	// If details is set to true, astronomer will print
	// additional factors such as percentiles.
	details bool
}

func buildRequestBody(ctx context, baseRequest string, pagination int) string {
	// Inject constant values into request body.
	requestBody := strings.Replace(baseRequest, "$repoOwner", ctx.repoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.repoName, 1)
	requestBody = strings.Replace(requestBody, "$pagination", fmt.Sprint(pagination), 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	return requestBody
}

// fetchStargazers fetches the list of cursors to iterate upon to
// fetch stargazer contributions.
func fetchStargazers(ctx context) (cursors []string, totalUsers uint, err error) {
	var (
		stargazers     []stargazers
		lastCursor     string
		page           int
		rateLimitSleep time.Duration
	)

	requestBody := buildRequestBody(ctx, fetchUsersRequest, listPagination)
	client := &http.Client{}

	defer disgo.EndStep()

	disgo.StartStep("Pre-fetching all stargazers")

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
			return nil, 0, disgo.FailStepf("unable to prepare request: %v", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.githubToken))
		req.Header.Set("User-Agent", "Astronomer")

		resp, err := getCache(ctx, req, listFilePagination(lastCursor))
		if err != nil {
			return nil, 0, disgo.FailStepf("unable to get cached file: %v", err)
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
			return nil, 0, fmt.Errorf("failed to fetch stargazers. last body recieved: %s", responseBody)
		}

		stargazers = append(stargazers, response.Repository.Stargazers)

		if len(response.Errors) != 0 || response.ErrorMessage != "" {
			disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
			return nil, 0, disgo.FailStepf("failed to fetch user contributions. last body recieved: %s", responseBody)
		}

		err = putCache(ctx, req, listFilePagination(lastCursor), responseBody)
		if err != nil {
			return nil, 0, disgo.FailStepf("unable to write user contribution data to cache: %v", err)
		}

		lastCursor = response.Repository.Stargazers.Meta.cursor()

		totalUsers += uint(len(response.Repository.Stargazers.Users))

		// TODO: This will break if the repository has a number of users that
		// is a factor of ${listPagination}. In that case, it will loop forever.
		// It would be nice to have a reliable way to figure out whether or not
		// we reached the end of the list.
		if len(response.Repository.Stargazers.Users) < listPagination {
			disgo.Infoln("Reached end of stargazer list")
			break
		}

		// Set the rate limit sleep duration depending on the token's limit.
		rateLimitSleepDuration = time.Hour / time.Duration(response.RateLimit.Limit)

		if response.RateLimit.Remaining <= 10 {
			disgo.Debugln("Rate limit reached, slowing down requests")
			rateLimitSleep = rateLimitSleepDuration
		}
	}

	cursors = getCursors(ctx, stargazers, totalUsers)

	return cursors, totalUsers, nil
}

// Return the appropriate cursors to be used by the fetchContributions function
// according to the value of ${contribPagination}. Also makes sure not to include
// any page of users containing blacklisted individuals.
func getCursors(ctx context, sg []stargazers, totalUsers uint) []string {
	var (
		skip      bool
		iteration uint
		cursors   []string
	)

	for _, stargazers := range sg {
		var currentPageUsers int

		for _, user := range stargazers.Users {
			if isBlacklisted(user.Login) {
				skip = true
			}

			// If this is the last user of the whole set, even if it's exactly at the
			// end of the current page, we don't need its cursor, because there is nothing
			// to get after his profile.
			if iteration == totalUsers-1 {
				break
			}

			// Iterate through list of stargazers, and add a cursor for every
			// ${contribPagination} users, unless one of the users within the current
			// page is blacklisted, in which case we skip the whole page.
			if iteration >= (contribPagination-1) && iteration%contribPagination == contribPagination-1 {
				if !skip {
					cursors = append(cursors, stargazers.Meta[currentPageUsers].Cursor)
				} else {
					skip = false
				}
			}

			iteration++
			currentPageUsers++
		}
	}

	if ctx.fastMode && totalUsers > ctx.stars {
		disgo.Infof("Fast mode enabled, scanning %d random stargazers out of %d\n", ctx.stars, totalUsers)
		return pickRandomStrings(cursors, ctx.stars/uint(contribPagination))
	}

	return cursors
}

// Pick random strings picks ${amount} random strings from the
// given slice of strings.
func pickRandomStrings(s []string, amount uint) []string {
	// Make the random non-deterministic.
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	var indexes []int
	for i := uint(1); i < amount; i++ {
		// Generate an index.
		newIndex := random.Intn(len(s) - 1)

		// Check if it has already been selected.
		var found bool
		for _, index := range indexes {
			if newIndex == index {
				found = true
			}
		}

		// Regenerate another one if this index has already been selected.
		if found {
			i--
			continue
		}

		indexes = append(indexes, newIndex)
	}

	var strings []string
	for _, index := range indexes {
		strings = append(strings, s[index])
	}

	// disgo.Infoln("Chosen cursors:", strings)

	return strings
}

func isBlacklisted(user string) bool {
	for _, blacklistedUser := range blacklistedUsers {
		if user == blacklistedUser {
			return true
		}
	}

	return false
}

func setupProgressBar(pages int) *mpb.Bar {
	p := mpb.New(mpb.WithWidth(64))

	bar := p.AddBar(int64(pages*contribPagination),
		mpb.BarStyle("[=>-]"),
		mpb.AppendDecorators(
			decor.Name("ETA: "),
			decor.AverageETA(decor.ET_STYLE_GO),
			decor.Name(" Elapsed: "),
			decor.Elapsed(decor.ET_STYLE_GO),
			decor.Name(" Progress: "),
			decor.Percentage()),
	)

	return bar
}

func getCursor(cursors []string, page int) string {
	if page > 1 {
		return cursors[page-2]
	}

	return "firstpage"
}

// fetchContributions fetches the contribution data of a list of stargazers.
// ctx contains the scanned context of the astronomer command.
// untilYear is the year until which to scan for contribuitons.
func fetchContributions(ctx context, cursors []string, untilYear int) ([]user, error) {
	var (
		users          []user
		rateLimitSleep time.Duration
	)

	requestBody := buildRequestBody(ctx, fetchContributionsRequest, contribPagination)
	client := &http.Client{}

	progressBar := setupProgressBar(len(cursors) + 1)

	// Iterate on pages of user contributions, following the cursors generated
	// in fetchStargazers.
	for page := 1; page <= len(cursors)+1; page++ {
		currentCursor := getCursor(cursors, page)

		// If this isn't the first page, inject the cursor value.
		paginatedRequestBody := requestBody
		if page > 1 {
			paginatedRequestBody = strings.Replace(
				paginatedRequestBody,
				fmt.Sprintf("stargazers(first:%d){", contribPagination),
				fmt.Sprintf("stargazers(first:%d,after:\\\"%s\\\"){", contribPagination, currentCursor),
				1)
		}

		// Get all user contributions for each year.
		currentYear := time.Now().Year()
		for i := 0; currentYear-i > untilYear-1; i++ {

			// Store start time in order to compute ETA for the progress bar.
			startTime := time.Now()

			// Inject the dates corresponding to the year we're scanning, into the request body.
			from := time.Date(currentYear-i, time.January, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(year - 1*time.Second)

			yearlyRequestBody := strings.Replace(paginatedRequestBody, "$dateFrom", from.Format(iso8601Format), 1)
			yearlyRequestBody = strings.Replace(yearlyRequestBody, "$dateTo", to.Format(iso8601Format), 1)

			// Prepare the HTTP request.
			req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer([]byte(yearlyRequestBody)))
			if err != nil {
				return nil, fmt.Errorf("unable to prepare request: %v", err)
			}

			// Inject GitHub token for API authorization.
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.githubToken))
			req.Header.Set("User-Agent", "Astronomer")

			// Try to get a cached response to this request.
			resp, err := getCache(ctx, req, contribFilePagination(currentCursor, currentYear-i))
			if err != nil {
				return nil, fmt.Errorf("unable to get cached file: %v", err)
			}

			response, responseBody, _ := parseResponse(resp)

			cachedFileFound := resp != nil

			// If the request was not found in the cache, try to fetch it until it works
			// or until the limit of 20 attempts is reached.
			if !cachedFileFound {
				// disgo.StartStepf("Fetching user contributions from users %d to %d for year %d", (page-1)*contribPagination, (page)*contribPagination, currentYear-i)

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
						return fmt.Errorf("unable to parse response: %v", err)
					}

					if len(response.Errors) != 0 || response.ErrorMessage != "" {
						if response.ErrorMessage != "" {
							return errors.New(response.ErrorMessage)
						}
						return errors.New(response.Errors[0].Message)
					}

					// If there is no error and no users in the response body, it must mean
					// that we reached the end of the user list.
					if len(response.Repository.Stargazers.Users) == 0 {
						resp.Body.Close()
						return nil
					}

					return nil
				}, backoff.NewConstantBackOff(15*time.Second))
			}

			if response == nil {
				return nil, fmt.Errorf("failed to fetch user contributions. failed at cursor %s", cursors[page-2])
			}

			// Update list of users with users from reponse.
			users = updateUsers(users, *response, currentYear-i)

			if len(response.Errors) != 0 || response.ErrorMessage != "" {
				disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
				return nil, fmt.Errorf("failed to fetch user contributions. failed at cursor %s", cursors[page-2])
			}

			// If file was fetched, write it in the cache. If we already got it from the cache,
			// no need to rewrite it.
			if !cachedFileFound {
				err = putCache(ctx, req, contribFilePagination(currentCursor, currentYear-i), responseBody)
				if err != nil {
					return users, fmt.Errorf("unable to write user contribution data to cache: %v", err)
				}
			}

			// If we approach the rate limit, slow the requests down.
			if response.RateLimit.Remaining <= 10 {
				disgo.Infoln("Rate limit reached, slowing down requests")
				rateLimitSleep = rateLimitSleepDuration
			}

			// Update progress bar.
			progressBar.IncrBy(contribPagination/(currentYear-untilYear), time.Since(startTime))
		}
	}

	return users, nil
}

// parseResponse parses a response from the GitHub API and converts it in the appropriate data model.
// It also returns the response body if it was read successfully.
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
		disgo.Errorf("Unable to parse body: %s\n", responseBody)
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

	// Otherwise, create the list of users and set their contributions appropriately.
	if !found {
		for idx := range newUsers {
			newUsers[idx].yearlyContributions = make(map[int]int)
			newUsers[idx].yearlyContributions[year] = newUsers[idx].Contributions.ContributionCalendar.TotalContributions + newUsers[idx].Contributions.PrivateContributions
		}

		users = append(users, newUsers...)
	}

	return users
}

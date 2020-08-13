package gql

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

	"github.com/Ullaakut/astronomer/pkg/context"
	"github.com/Ullaakut/disgo"
	"github.com/Ullaakut/disgo/style"
	"github.com/cenkalti/backoff/v3"
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
		// "jstrachan", // has been fixed.
	}
)

// FetchStargazers fetches the list of cursors to iterate upon to
// fetch stargazer contributions.
func FetchStargazers(ctx *context.Context) (cursors []string, totalUsers uint, err error) {
	var (
		stargazers     []stargazers
		lastCursor     string
		page           int
		rateLimitSleep time.Duration
	)

	if ctx.Stars < uint(contribPagination) {
		return nil, 0, fmt.Errorf("unable to compute less stars than the amount fetched per page. Please set stars to at least %d", contribPagination)
	}

	// Round amount of stars to get according to pagination.
	if ctx.Stars%contribPagination != 0 {
		ctx.Stars = ctx.Stars - ctx.Stars%contribPagination
		disgo.Errorln(style.Failure("Rounding amount of stars to fetch to ", ctx.Stars, " in order to match pagination"))
	}

	// Inject constants in request body.
	requestBody := buildRequestBody(ctx, fetchUsersRequest, listPagination)
	client := &http.Client{}

	disgo.StartStep("Pre-fetching all stargazers")

	defer disgo.EndStep()

	for {
		var response *listStargazersResponse

		page++

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

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.GithubToken))
		req.Header.Set("User-Agent", "Astronomer")

		// Attempt to find the response to this specific request already stored
		// in the cache directory.
		resp, err := getCache(ctx, req, listFilePagination(lastCursor))
		if err != nil {
			return nil, 0, disgo.FailStepf("unable to get cached file: %v", err)
		}

		response, responseBody, _ := parseResponse(resp)

		// If the request was not found in the cache, try to fetch it until it works
		// or until the limit of 20 attempts is reached.
		if resp == nil {
			var attempts int
			err = backoff.Retry(func() error {
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

		if response == nil || err != nil {
			return nil, 0, fmt.Errorf("failed to fetch stargazers. last body recieved: %s", responseBody)
		}

		stargazers = append(stargazers, response.Repository.Stargazers)

		if len(response.Errors) != 0 || response.ErrorMessage != "" {
			disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
			return nil, 0, disgo.FailStepf("failed to fetch user contributions. last body recieved: %s", responseBody)
		}

		// Since we arrived here, we got a successful response, so we store it
		// in the cache directory.
		err = putCache(ctx, req, listFilePagination(lastCursor), responseBody)
		if err != nil {
			return nil, 0, disgo.FailStepf("unable to write user contribution data to cache: %v", err)
		}

		lastCursor = response.Repository.Stargazers.Meta.cursor()

		totalUsers += uint(len(response.Repository.Stargazers.Users))

		if len(response.Repository.Stargazers.Users) < listPagination {
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

// FetchContributions fetches the contribution data of a list of stargazers.
// ctx contains the scanned context of the astronomer command.
// untilYear is the year until which to scan for contribuitons.
func FetchContributions(ctx *context.Context, cursors []string, untilYear int) ([]User, error) {
	var (
		users          []User
		rateLimitSleep time.Duration
	)

	requestBody := buildRequestBody(ctx, fetchContributionsRequest, contribPagination)
	client := &http.Client{}

	progress, bar := setupProgressBar(len(cursors))
	defer progress.Wait()

	// If we are scanning only a portion of stargazers, the
	// scan does not start with a page without a cursor.
	isReverseOrder := uint(len(cursors)) > ctx.Stars/contribPagination

	totalPages := len(cursors)

	// If we don't scan in reverse order (first stars first), we
	// have fetch each page pointed at by the cursors, plus the first
	// page which doesn't require a cursor.
	if !isReverseOrder {
		totalPages++
	}

	// Iterate on pages of user contributions, following the cursors generated
	// in fetchStargazers.
	for page := 1; page <= totalPages; page++ {
		currentCursor := getCursor(cursors, page, isReverseOrder)

		// If this isn't the first page, inject the cursor value.
		paginatedRequestBody := requestBody
		if currentCursor != "firstpage" {
			paginatedRequestBody = strings.Replace(
				paginatedRequestBody,
				fmt.Sprintf("stargazers(first:%d){", contribPagination),
				fmt.Sprintf("stargazers(first:%d,after:\\\"%s\\\"){", contribPagination, currentCursor),
				1,
			)
		}

		// Get all user contributions for each year.
		currentYear := time.Now().Year()
		for i := 0; currentYear-i > untilYear-1; i++ {
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
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ctx.GithubToken))
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
				var attempts int
				err = backoff.Retry(func() error {
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

			if response == nil || err != nil {
				return nil, fmt.Errorf("failed to fetch user contributions. failed at cursor %s", currentCursor)
			}

			// Update list of users with users from reponse.
			users = updateUsers(users, *response, currentYear-i)

			if len(response.Errors) != 0 || response.ErrorMessage != "" {
				disgo.Errorln("Errors:", response.ErrorMessage, response.Errors)
				return nil, fmt.Errorf("failed to fetch user contributions. failed at cursor %s", currentCursor)
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
			bar.IncrBy(contribPagination / (currentYear - untilYear))
		}
	}

	bar.Abort(true)

	return users, nil
}

func buildRequestBody(ctx *context.Context, baseRequest string, pagination int) string {
	// Inject constant values into request body.
	requestBody := strings.Replace(baseRequest, "$repoOwner", ctx.RepoOwner, 1)
	requestBody = strings.Replace(requestBody, "$repoName", ctx.RepoName, 1)
	requestBody = strings.Replace(requestBody, "$pagination", fmt.Sprint(pagination), 1)

	// Remove all `\n` so that it's valid JSON. Remove all spaces.
	requestBody = strings.Replace(requestBody, "\t", "", -1)
	requestBody = strings.Replace(requestBody, " ", "", -1)
	requestBody = strings.Replace(requestBody, "\n", " ", -1)

	return requestBody
}

// Return the appropriate cursors to be used by the fetchContributions function
// according to the value of ${contribPagination}. Also makes sure not to include
// any page of users containing blacklisted individuals.
func getCursors(ctx *context.Context, sg []stargazers, totalUsers uint) []string {
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

	if totalUsers <= 219 {
		disgo.Infof("All %d stargazers will be scanned\n", totalUsers)
		return cursors
	}

	var selectedCursors []string

	// totalCursorAmount is the total amount of cursors to fetch.
	totalCursorAmount := int(ctx.Stars) / contribPagination

	// beginCursorAmount is the amount of cursors to fetch for the 200 first users.
	disgo.Infof("Selecting 200 first stargazers out of %d\n", totalUsers)
	beginCursorAmount := 200/contribPagination - 1

	selectedCursors = append(selectedCursors, cursors[len(cursors)-beginCursorAmount-1:len(cursors)-1]...)

	if ctx.ScanAll || totalUsers < ctx.Stars {
		disgo.Infof("Selecting all %d remaining stargazers\n", totalUsers-200)
		selectedCursors = append(selectedCursors, cursors[:len(cursors)-beginCursorAmount]...)
	} else {
		// endCursorAmount is the amount of cursors to fetch to get the random users.
		endCursorAmount := totalCursorAmount - beginCursorAmount
		disgo.Infof("Selecting %d random stargazers out of %d\n", (endCursorAmount-1)*contribPagination, totalUsers)

		selectedCursors = pickRandomStringsExcept(cursors, selectedCursors, uint(endCursorAmount))
	}

	return selectedCursors
}

// Pick random strings picks ${amount} random strings from the
// given slice of strings, except those that were already picked.
func pickRandomStringsExcept(s []string, picked []string, amount uint) []string {
	// Make the random non-deterministic.
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	for i := uint(1); i < amount; i++ {
		// Pick a string.
		newPick := s[random.Intn(len(s)-1)]

		// Check if it has already been selected.
		var found bool
		for _, alreadyPicked := range picked {
			if newPick == alreadyPicked {
				found = true
			}
		}

		// Regenerate another one if this index has already been selected.
		if found {
			i--
			continue
		}

		picked = append(picked, newPick)
	}

	return picked
}

// isBlacklisted checks if a user is blacklisted.
func isBlacklisted(user string) bool {
	for _, blacklistedUser := range blacklistedUsers {
		if user == blacklistedUser {
			return true
		}
	}

	return false
}

// setupProgressBar sets the progress bar properly according to
// the expected amount of pages of data.
func setupProgressBar(pages int) (*mpb.Progress, *mpb.Bar) {
	p := mpb.New(mpb.WithWidth(64))

	bar := p.AddBar(int64(pages*contribPagination),
		mpb.BarRemoveOnComplete(),
		mpb.AppendDecorators(
			decor.Name("ETA: "),
			decor.AverageETA(decor.ET_STYLE_GO),
			decor.Name(" Elapsed: "),
			decor.Elapsed(decor.ET_STYLE_GO),
			decor.Name(" Progress: "),
			decor.Percentage()),
	)

	return p, bar
}

// getCursor returns the current cursor for the given page, depending on the
// order the cursors are being read in.
func getCursor(cursors []string, page int, reverseOrder bool) string {
	// If scanning in the reverse order, we don't have any page without
	// a cursor, so we don't start using the cursor from page 2 but
	// the first one directly.
	if reverseOrder && page > 0 {
		return cursors[page-1]
	}

	// If not scanning in the reverse order, the first page does not
	// need a cursor since we can simply request the first X users.
	if page > 1 {
		return cursors[page-2]
	}

	return "firstpage"
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
		return nil, responseBody, fmt.Errorf("error while querying user data: %v [%s:%s]", response.Errors[0].Message, response.Errors[0].Extensions.ArgumentName, response.Errors[0].Extensions.Name)
	}

	return &response, responseBody, nil
}

// updateUsers updates a slice of user from the data in a list stargazer response.
// It also sets their yearly contributions accordingly.
func updateUsers(users []User, response listStargazersResponse, year int) []User {
	var (
		found    bool
		newUsers []User
	)

	newUsers = response.Repository.Stargazers.Users

	// Update users if they already exist in the list.
	for idx := range users {
		for _, u := range newUsers {
			if users[idx].Login == u.Login {
				users[idx].YearlyContributions[year] = u.Contributions.ContributionCalendar.TotalContributions + u.Contributions.PrivateContributions

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
			newUsers[idx].YearlyContributions = make(map[int]int)
			newUsers[idx].YearlyContributions[year] = newUsers[idx].Contributions.ContributionCalendar.TotalContributions + newUsers[idx].Contributions.PrivateContributions
		}

		users = append(users, newUsers...)
	}

	return users
}

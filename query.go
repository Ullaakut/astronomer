package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ullaakut/disgo"
)

const (
	githubAPI        = "https://api.github.com/"
	contributionsAPI = "https://github-contributions-api.now.sh/v1/"
)

// Context holds config information used to query GitHub.
type Context struct {
	Repo     string // Repository (owner/repository)
	Token    string // Access token
	CacheDir string // Cache directory

	acceptHeader string // Optional Accept: header value
}

// User represents a GitHub user.
type User struct {
	Login            string `json:"login"`
	ID               int    `json:"id"`
	AvatarURL        string `json:"avatar_url"`
	GravatarID       string `json:"gravatar_id"`
	URL              string `json:"url"`
	HTMLURL          string `json:"html_url"`
	FollowersURL     string `json:"followers_url"`
	FollowingURL     string `json:"following_url"`
	StarredURL       string `json:"starred_url"`
	SubscriptionsURL string `json:"subscriptions_url"`
	Type             string `json:"type"`
	SiteAdmin        bool   `json:"site_admin"`
	Name             string `json:"name"`
	Company          string `json:"company"`
	Blog             string `json:"blog"`
	Location         string `json:"location"`
	Email            string `json:"email"`
	Hireable         bool   `json:"hireable"`
	Bio              string `json:"bio"`
	PublicRepos      int    `json:"public_repos"`
	PublicGists      int    `json:"public_gists"`
	Followers        int    `json:"followers"`
	Following        int    `json:"following"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`

	Contributions   Contributions `json:"contributions"`
	Trustworthiness float64       `json:"trustworthiness"`
}

// Contributions represents all of a user's lifetime contributions.
type Contributions struct {
	Years []YearlyContribution `json:"years"`
}

// YearlyContribution represents the user's total contributions per year.
type YearlyContribution struct {
	Year  string `json:"year"`
	Total int    `json:"total"`
}

// Age returns the age (time from current time to created at
// timestamp) of this stargazer in days.
func (s User) Age() (float64, error) {
	curDay := time.Now().Unix()
	createT, err := time.Parse(time.RFC3339, s.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("failed to parse created at timestamp (%s): %s", s.CreatedAt, err)
	}

	// Return the difference in days between the account creation and today.
	return float64((curDay - createT.Unix()) / 3600 / 24), nil
}

// getAllUsers recursively descends into GitHub API endpoints, starting
// with the list of stargazers for the repo.
func getAllUsers(ctx *Context) error {
	disgo.StartStep("Fetching list of stargazers from GitHub API")
	sg, err := queryStargazers(ctx)
	if err != nil {
		return disgo.FailStepf("unable to fetch stargazers: %v", err)
	}

	disgo.Infof("Fetched %d stargazers", len(sg))

	disgo.StartStep("Querying user info from each stargazer")
	if err = queryUserInfo(ctx, sg); err != nil {
		return disgo.FailStepf("unable to fetch user info: %v", err)
	}

	disgo.StartStep("Querying user contributions from each stargazer")
	if err = queryUserContributions(ctx, sg); err != nil {
		return disgo.FailStepf("unable to fetch contributions: %v", err)
	}

	return saveState(ctx, sg)
}

// queryStargazers queries the repo's stargazers API endpoint.
// Returns the complete slice of stargazers.
func queryStargazers(ctx *Context) ([]User, error) {
	ctxCopy := *ctx
	ctxCopy.acceptHeader = "application/vnd.github.v3.star+json"
	url := fmt.Sprintf("%srepos/%s/stargazers", githubAPI, ctx.Repo)

	var (
		stargazers []User
		err        error
	)

	disgo.Debugf("Querying stargazers of repository %s", ctx.Repo)

	// Get each item in the list of stargazers and append them to
	// our slice.
	for len(url) > 0 {
		var fetched []User
		url, err = fetchURL(&ctxCopy, url, &fetched, true)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch stargazer list from URL %q: %v", url, err)
		}

		stargazers = append(stargazers, fetched...)
	}

	return stargazers, nil
}

// queryUserInfo queries user info for each stargazer.
func queryUserInfo(ctx *Context, sg []User) error {
	for idx := range sg {
		if _, err := fetchURL(ctx, sg[idx].URL, &sg[idx], false); err != nil {
			return fmt.Errorf("unable to fetch user info from URL %q: %v", sg[idx].URL, err)
		}
	}

	return nil
}

// queryUserContributions queries all contributions to subscribed repos
// for each stargazer.
func queryUserContributions(ctx *Context, sg []User) error {
	for idx := range sg {
		url := fmt.Sprint(contributionsAPI, sg[idx].Login)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("unable to fetch user contributions from URL %q: %v", url, err)
		}

		resp, err := getCache(ctx, req)
		if err != nil {
			return fmt.Errorf("unable to get cache from URL %q: %v", req.URL, err)
		}

		if resp == nil {
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				disgo.Errorf("Unable to get contributions for user %v: %v\n", sg[idx], err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				disgo.Errorf("Unable to get contributions for user %v: %d %v\n", sg[idx], resp.StatusCode, resp.Status)
				continue
			}

			if err := putCache(ctx, req, resp); err != nil {
				disgo.Errorf("Unable to cache contributions for user %q: %v\n", sg[idx].Login, err)
			}
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			disgo.Errorf("Unable to read response from GitHub Contributions API: %v\n", err)
			continue
		}

		if err := json.Unmarshal(data, &sg[idx].Contributions); err != nil {
			disgo.Errorf("Unable to decode response from GitHub Contributions API: %v", err)
		}
	}

	return nil
}

// saveState writes all queried stargazer and repo data.
func saveState(ctx *Context, sg []User) error {
	filename := filepath.Join(ctx.CacheDir, ctx.Repo, "saved_state")

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(sg); err != nil {
		return fmt.Errorf("failed to encode stargazer data: %s", err)
	}

	return nil
}

// loadState reads previously saved queried stargazer and repo data.
func loadState(ctx *Context) ([]User, error) {
	filename := filepath.Join(ctx.CacheDir, ctx.Repo, "saved_state")

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sg []User
	dec := json.NewDecoder(f)
	if err := dec.Decode(&sg); err != nil {
		return nil, fmt.Errorf("failed to decode stargazer data: %s", err)
	}

	return sg, nil
}

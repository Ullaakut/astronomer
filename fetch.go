package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ullaakut/disgo"
)

const (
	millisecond = 1000000
	second      = 1000000000
)

// A rateLimitError is returned when the requestor's rate limit has
// been exceeded. It allows us to know for how long to wait before
// making more requests.
type rateLimitError struct {
	resetUnix int64 // Unix seconds at which rate limit is reset.
}

// Error implements the error interface.
func (r *rateLimitError) Error() string {
	reset := time.Unix(r.resetUnix, 0).Local()

	return fmt.Sprintf("Rate limit resets at %s (in %s)", reset, r.expiration())
}

// expiration returns the duration until the rate limit expires,
// including 10 seconds of padding to account for clock offset.
func (r *rateLimitError) expiration() time.Duration {
	return time.Unix(r.resetUnix, 0).Add(10 * time.Second).Sub(time.Now())
}

// An httpError specifies a non-200 HTTP response code.
type httpError struct {
	req  *http.Request
	resp *http.Response
}

// Error implements the error interface.
func (e *httpError) Error() string {
	return fmt.Sprintf("failed to fetch %q: %v", e.req.URL, e.resp)
}

// linkRegexp is used for parsing the "Link" HTTP header.
var linkRegexp = regexp.MustCompile(`^<(.*)>; rel="next", <(.*)>; rel="last".*`)

// fetchURL fetches the specified URL. The cache (specified in
// c.CacheDir) is consulted first and if not found, the specified URL
// is fetched using the HTTP default client. The refresh bool indicates
// whether the last page of results should be refreshed if it's found
// in the response cache. Returns the next URL if the result is paged
// or an error on failure.
//
// If it finds paged results, it calls itself recursively until the last page.
func fetchURL(c *Context, url string, value interface{}, refresh bool) (string, error) {
	// Remove whitespaces, and trim before and after `<` and `>`.
	url = strings.Split(strings.TrimPrefix(strings.TrimSpace(url), "<"), ">")[0]

	// Create request and add mandatory user agent and accept encoding headers.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Only use the token on the GitHub API.
	if strings.Contains(url, "https://api.github.com/") {
		req.Header.Add("User-Agent", "GH Fake Stars Detector")
		req.Header.Add("Authorization", fmt.Sprintf("token %s", c.Token))
		req.Header.Add("Accept-Encoding", "application/json")
	}

	// Look for a cached response.
	var (
		resp   *http.Response
		next   string
		cached = true
	)

	resp, err = getCache(c, req)
	if err != nil {
		return "", nil
	}

	// Loop until a next URL is found or a direct result was received.
	// The last page is always checked in case it changed since the last scan.
	for {
		// If not found, fetch the URL from the GitHub API server.
		if resp == nil {
			cached = false
			// Retry up to 10 times.
			for i := uint(0); i < 10; i++ {
				resp, err = doFetch(c, url, req)
				if err == nil {
					break
				}

				switch t := err.(type) {
				case *rateLimitError:
					// Sleep until the expiration of the rate limit.
					disgo.Errorf("Rate limit error, waiting: %s\n", t)
					time.Sleep(t.expiration())
				case *httpError:
					disgo.Errorf("error while fetching %q: %v", url, err.(*httpError).resp)
					return "", nil
				default:
					// Retry with exponential backoff on random connection and networking errors.
					disgo.Errorf("Potential network error: %s\n", t)

					// Backoff increases with each attempt.
					backoff := int64((1 << i)) * 50 * millisecond

					// Limit backoff to 1 second.
					if backoff > 1*second {
						backoff = 1 * second
					}

					time.Sleep(time.Duration(backoff))
				}
			}
		}

		if resp == nil {
			return "", nil
		}

		// Parse the next link, if available.
		if link := resp.Header.Get("Link"); len(link) > 0 {
			urls := linkRegexp.FindStringSubmatch(link)
			var found bool
			if len(urls) != 0 {
				urls = strings.Split(urls[0], ",")
				for _, url := range urls {
					if strings.Contains(url, "rel=\"next\"") {
						next = url
						found = true
						break
					}
				}

				if !found {
					next = urls[1]
				}
			}
		}

		// If this is the last page of a cached response, re-fetch it.
		if len(next) > 0 || !cached || !refresh {
			break
		}

		resp.Body.Close() // Don't forget to close the body
		resp = nil
	}

	var body []byte
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		// If cache entry is unreadable, re-fetch it from its URL.
		disgo.Errorf("Cache entry %q corrupted; removing and refetching\n", url)
		clearEntry(c, url)
		return fetchURL(c, url, value, refresh)
	}

	if err = json.Unmarshal(body, value); err != nil {
		return "", fmt.Errorf("unable to unmarshal response from %q: %v", url, err)
	}

	return next, nil
}

// doFetch performs the GET HTTPS request and stores the result in the
// cache on success. A rateLimitError is returned in the event that
// the access token has exceeded its hourly limit (usually 5000 requests).
func doFetch(c *Context, url string, req *http.Request) (*http.Response, error) {
	disgo.Debugf("Fetching %q\n", url)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while fetching %q: %v", url, err)
	}

	switch resp.StatusCode {
	case 200:
		if err := putCache(c, req, resp); err == nil {
			return resp, nil
		}

	case 202: // Request was accepted but needs to be retried after a short backoff.
		err = errors.New("202 (Accepted) HTTP response: retrying")

	case 403: // Forbidden / Rate limiting
		if limitRem := resp.Header.Get("X-rateLimit-Remaining"); len(limitRem) > 0 {
			var remaining, resetUnix int
			if remaining, err = strconv.Atoi(limitRem); err == nil && remaining == 0 {
				if limitReset := resp.Header.Get("X-rateLimit-Reset"); len(limitReset) > 0 {
					if resetUnix, err = strconv.Atoi(limitReset); err == nil {
						err = &rateLimitError{resetUnix: int64(resetUnix)}
						resp.Body.Close()
						return nil, err
					}
				}
			}
		}
		err = &httpError{req, resp}

	default:
		err = &httpError{req, resp}
	}

	resp.Body.Close()
	return nil, err
}

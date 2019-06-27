package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kennygrant/sanitize"
)

// getCache searches the cache directory for a file matching the
// supplied request's URL. If found, the file contains a cached copy
// of the HTTP response. The contents are read into an http.Response
// object and returned.
func getCache(c *Context, req *http.Request) (*http.Response, error) {
	filename := cacheEntryFilename(c, req.URL.String())
	pathToCreate := path.Dir(filename)

	if err := os.MkdirAll(pathToCreate, os.ModeDir|0755); err != nil {
		return nil, fmt.Errorf("unable to create path %q: %v", pathToCreate, err)
	}

	resp, err := readCachedResponse(filename, req)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to read cached response: %v", err)
	}

	return resp, nil
}

func readCachedResponse(filename string, req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read cached response from file %q: %v", filename, err)
	}

	return http.ReadResponse(bufio.NewReader(bytes.NewBuffer(body)), req)
}

// putCache puts the supplied http.Response into the cache.
func putCache(c *Context, req *http.Request, resp *http.Response) error {
	defer resp.Body.Close()

	filename := cacheEntryFilename(c, req.URL.String())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create cache file: %v", err)
	}

	if err := resp.Write(f); err != nil {
		f.Close()
		return fmt.Errorf("unable to write in cache: %v", err)
	}

	f.Close()

	readResp, err := readCachedResponse(filename, req)
	if err != nil {
		return fmt.Errorf("unable to read cached response: %v", err)
	}

	resp.Body = readResp.Body
	return nil
}

// cacheEntryFilename creates a filename-safe name in a subdirectory
// of the configured cache dir, with any access token stripped out.
func cacheEntryFilename(c *Context, url string) string {
	newURL := strings.Replace(url, fmt.Sprintf("access_token=%s", c.Token), "", 1)
	return filepath.Join(c.CacheDir, c.Repo, sanitize.BaseName(newURL))
}

// clearEntry clears a specified cache entry.
func clearEntry(c *Context, url string) error {
	filename := cacheEntryFilename(c, url)
	return os.Remove(filename)
}

// Clear clears all cache entries for the repository specified in the
// fetch context.
func Clear(c *Context) error {
	filename := filepath.Join(c.CacheDir, c.Repo)
	return os.RemoveAll(filename)
}

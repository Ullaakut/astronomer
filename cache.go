package main

import (
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
func getCache(ctx context, req *http.Request, pagination string) (*http.Response, error) {
	filename := cacheEntryFilename(ctx, req.URL.String()+pagination)
	pathToCreate := path.Dir(filename)

	if err := os.MkdirAll(pathToCreate, os.ModeDir|0755); err != nil {
		return nil, err
	}

	resp, err := readCachedResponse(filename, req)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	return resp, nil
}

func readCachedResponse(filename string, req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(body)),
	}, nil
}

// putCache puts the supplied http.Response into the cache.
func putCache(ctx context, req *http.Request, pagination string, body []byte) error {
	filename := cacheEntryFilename(ctx, req.URL.String()+pagination)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create cache file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(body)
	if err != nil {
		return fmt.Errorf("unable to write response in cache file: %v", err)
	}

	_, err = readCachedResponse(filename, req)
	if err != nil {
		return err
	}

	return nil
}

// cacheEntryFilename creates a filename-safe name in a subdirectory
// of the configured cache dir, with any access token stripped out.
func cacheEntryFilename(ctx context, url string) string {
	newURL := strings.Replace(url, fmt.Sprintf("access_token=%s", ctx.githubToken), "", 1)
	return filepath.Join(ctx.cacheDirectoryPath, ctx.repoOwner, ctx.repoName, sanitize.BaseName(newURL))
}

// clearEntry clears a specified cache entry.
func clearEntry(ctx context, url string) error {
	filename := cacheEntryFilename(ctx, url)
	return os.Remove(filename)
}

// Clear clears all cache entries for the repository specified in the
// fetch context.
func Clear(ctx context) error {
	filename := filepath.Join(ctx.cacheDirectoryPath, ctx.repoOwner, ctx.repoName)
	return os.RemoveAll(filename)
}

// listilePagination generates the pagination to append to the cache file names
// for stargazer lists.
func listFilePagination(page int) string {
	return fmt.Sprintf("-list-%d", page)
}

// contribFilePagination generates the pagination to append to the cache file names
// for user contribution data.
func contribFilePagination(page, year int) string {
	return fmt.Sprintf("-%d-%d", page, year)
}

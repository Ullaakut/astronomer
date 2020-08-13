package gql

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
	"github.com/Ullaakut/astronomer/pkg/context"
)

// getCache searches the cache directory for a file matching the
// supplied request's URL. If found, the file contains a cached copy
// of the HTTP response. The contents are read into an http.Response
// object and returned.
func getCache(ctx *context.Context, req *http.Request, pagination string) (*http.Response, error) {
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
func putCache(ctx *context.Context, req *http.Request, pagination string, body []byte) error {
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
func cacheEntryFilename(ctx *context.Context, url string) string {
	newURL := strings.Replace(url, fmt.Sprintf("access_token=%s", ctx.GithubToken), "", 1)
	return filepath.Join(ctx.CacheDirectoryPath, ctx.RepoOwner, ctx.RepoName, sanitize.BaseName(newURL))
}

// listilePagination generates the pagination to append to the cache file names
// for stargazer lists.
func listFilePagination(cursor string) string {
	if cursor == "" {
		return fmt.Sprintf("-list-firstpage")
	}

	return fmt.Sprintf("-list-%s", cursor)
}

// contribFilePagination generates the pagination to append to the cache file names
// for user contribution data.
func contribFilePagination(cursor string, year int) string {
	if cursor == "" {
		return fmt.Sprintf("-firstpage-%d", year)
	}

	return fmt.Sprintf("-%s-%d", cursor, year)
}

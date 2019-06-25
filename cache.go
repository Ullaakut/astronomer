// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Spencer Kimball (spencer.kimball@gmail.com)

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
	if err := os.MkdirAll(path.Dir(filename), os.ModeDir|0755); err != nil {
		return nil, err
	}

	resp, err := readCachedResponse(filename, req)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	return resp, err
}

func readCachedResponse(filename string, req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewBuffer(body)), req)
}

// putCache puts the supplied http.Response into the cache.
func putCache(c *Context, req *http.Request, resp *http.Response) error {
	defer resp.Body.Close()
	filename := cacheEntryFilename(c, req.URL.String())
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	if err := resp.Write(f); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// TODO(spencer): this sucks, but we must re-read the response as
	// the body is closed during the call to resp.Write().
	if readResp, err := readCachedResponse(filename, req); err != nil {
		return err
	} else {
		resp.Body = readResp.Body
	}
	return nil
}

// cacheEntryFilename creates a filename-safe name in a subdirectory
// of the configured cache dir, with any access token stripped out.
func cacheEntryFilename(c *Context, url string) string {
	newUrl := strings.Replace(url, fmt.Sprintf("access_token=%s", c.Token), "", 1)
	return filepath.Join(c.CacheDir, c.Repo, sanitize.BaseName(newUrl))
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

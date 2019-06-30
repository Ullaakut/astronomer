package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheEntryFilename(t *testing.T) {
	ctx := context{
		repoOwner:          "ullaakut",
		repoName:           "astronomer",
		githubToken:        "fakeToken",
		cacheDirectoryPath: "./data",
	}

	sanitizedFilename := cacheEntryFilename(ctx, "https://fakeapi.com/graphql?access_token=fakeToken-1-2019")

	assert.Equal(t, "data/ullaakut/astronomer/https-fakeapi-com-graphql-1-2019", sanitizedFilename)
}

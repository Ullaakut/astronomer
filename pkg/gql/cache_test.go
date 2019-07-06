package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ullaakut/astronomer/pkg/context"
)

func TestCacheEntryFilename(t *testing.T) {
	ctx := &context.Context{
		RepoOwner:          "ullaakut",
		RepoName:           "astronomer",
		GithubToken:        "fakeToken",
		CacheDirectoryPath: "./data",
	}

	sanitizedFilename := cacheEntryFilename(ctx, "https://fakeapi.com/graphql?access_token=fakeToken-1-2019")

	assert.Equal(t, "data/ullaakut/astronomer/https-fakeapi-com-graphql-1-2019", sanitizedFilename)
}

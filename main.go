package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

func main() {
	args := os.Args[1:]

	if len(args) != 1 {
		disgo.Errorln(style.Failure(style.SymbolCross, " Invalid number of arguments. Only argument should be the repository name (owner/repository)"))
		os.Exit(1)
	}

	if err := detectFakeStars(args[0]); err != nil {
		disgo.Errorln(style.Failure(style.SymbolCross, " ", err))
		os.Exit(1)
	}
}

func detectFakeStars(repository string) error {
	disgo.SetTerminalOptions(disgo.WithColors(true), disgo.WithDebug(true))

	repoInfo := strings.Split(repository, "/")
	if len(repoInfo) != 2 {
		return fmt.Errorf("invalid repository %q: should be of the form \"repoOwner/repoName\"", repository)
	}

	ctx := context{
		repoOwner:          repoInfo[0],
		repoName:           repoInfo[1],
		githubToken:        os.Getenv("GITHUB_TOKEN"),
		cacheDirectoryPath: "./data",
		scanUntilYear:      2013,
	}

	disgo.Infof("Beginning fetching process for repository %q\n", repository)
	users, err := fetchStargazers(ctx)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	if len(users) < 300 {
		disgo.Infoln(style.Important("This repository appears to have a low amount of stargazers. Trust calculations might not be accurate."))
	}

	report, err := computeTrustReport(users)
	if err != nil {
		disgo.Infof("%+v\n", report)
		return fmt.Errorf("failed to analyze stargazer data: %v", err)
	}

	renderReport(report)

	disgo.Infof("%s Analysis successful. %d users computed.\n", style.Success(style.SymbolCheck), len(users))

	return nil
}

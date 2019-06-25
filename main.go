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

	ctx := &Context{
		Repo:     strings.ToLower(repository),
		Token:    os.Getenv("GITHUB_TOKEN"),
		CacheDir: "./data",
	}

	disgo.Infof("Beginning fetching process for repository %q\n", repository)
	if err := getAllUsers(ctx); err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	disgo.Infof("Loading state")
	stargazers, err := loadState(ctx)
	if err != nil {
		return fmt.Errorf("failed to load saved stargazer data: %s", err)
	}

	if len(stargazers) < 300 {
		disgo.Infoln(style.Important("This repository appears to have a low amount of stargazers. Trust calculations might not be accurate."))
	}

	disgo.Infof("Computing trust factors")
	report, err := computeTrustReport(stargazers)
	if err != nil {
		return fmt.Errorf("failed to analyze stargazer data: %s", err)
	}

	renderReport(report)

	disgo.Infof("%s Analysis successful. %d users computed.\n", style.Success(style.SymbolCheck), len(stargazers))

	return nil
}

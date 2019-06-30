package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

func parseArguments() error {
	viper.SetEnvPrefix("astronomer")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.BoolP("details", "d", false, "Show more detailed trust factors, such as percentiles")
	pflag.StringP("cachedir", "c", "./data", "Set the directory in which to store cache data")

	viper.AutomaticEnv()

	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}

	if viper.GetBool("help") || len(pflag.Args()) == 0 {
		disgo.Infoln("Missing required repository argument")
		pflag.Usage()
		os.Exit(0)
	}

	return nil
}

func main() {
	disgo.SetTerminalOptions(disgo.WithColors(true), disgo.WithDebug(true))

	err := parseArguments()
	if err != nil {
		disgo.Errorln(style.Failure(style.SymbolCross, err))
		os.Exit(1)
	}

	repository := pflag.Arg(0)

	repoInfo := strings.Split(repository, "/")
	if len(repoInfo) != 2 {
		disgo.Errorln(style.Failure(style.SymbolCross, "invalid repository %q: should be of the form \"repoOwner/repoName\"", repository))
		os.Exit(1)
	}

	ctx := context{
		repoOwner:          repoInfo[0],
		repoName:           repoInfo[1],
		githubToken:        os.Getenv("GITHUB_TOKEN"),
		cacheDirectoryPath: viper.GetString("cachedir"),
		details:            viper.GetBool("details"),
	}

	if err := detectFakeStars(ctx); err != nil {
		disgo.Errorln(style.Failure(style.SymbolCross, " ", err))
		os.Exit(1)
	}
}

func detectFakeStars(ctx context) error {
	disgo.SetTerminalOptions(disgo.WithColors(true), disgo.WithDebug(true))

	disgo.Infof("Beginning fetching process for repository %s/%s\n", ctx.repoOwner, ctx.repoName)
	cursors, err := fetchStargazers(ctx)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	if len(cursors) < 300/contribPagination {
		disgo.Infoln(style.Important("This repository appears to have a low amount of stargazers. Trust calculations might not be accurate."))
	}

	users, err := fetchContributions(ctx, cursors, 2013)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	report, err := computeTrustReport(ctx, users)
	if err != nil {
		disgo.Infof("%+v\n", report)
		return fmt.Errorf("failed to analyze stargazer data: %v", err)
	}

	renderReport(ctx.details, report)

	disgo.Infof("%s Analysis successful. %d users computed.\n", style.Success(style.SymbolCheck), len(users))

	return nil
}

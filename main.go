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

	pflag.BoolP("fast", "f", true, "Enable fast mode in order to scan random stargazers instead of all of them (slightly less accurate)")
	pflag.UintP("stars", "s", 1000, "Maxmimum amount of stars to scan, if fast mode is enabled")
	pflag.BoolP("scanfirststars", "", false, "Scan the first stars of the repository (overrides fast mode). Set amount of stars with --stars options")
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
		disgo.Errorln(style.Failure(style.SymbolCross, " invalid repository %q: should be of the form \"repoOwner/repoName\"", repository))
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		disgo.Errorln(style.Failure(style.SymbolCross, " missing github access token. Please set one in your GITHUB_TOKEN environment variable, with \"repo\" rights."))
		os.Exit(1)
	}

	stars := viper.GetUint("stars")
	if stars < uint(contribPagination) {
		disgo.Errorln(style.Failure(style.SymbolCross, " unable to compute less stars than the amount fetched per page. Please set stars to at least ", contribPagination))
		os.Exit(1)
	}

	if stars%contribPagination != 0 {
		stars = stars - stars%contribPagination
		disgo.Errorln(style.Failure("Rounding amount of stars to get to ", stars, " instead of ", viper.GetUint("stars"), " to match pagination"))
	}

	ctx := context{
		repoOwner:          repoInfo[0],
		repoName:           repoInfo[1],
		githubToken:        token,
		cacheDirectoryPath: viper.GetString("cachedir"),
		fastMode:           viper.GetBool("fast"),
		scanFirstStars:     viper.GetBool("scanfirststars"),
		stars:              stars,
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
	cursors, totalUsers, err := fetchStargazers(ctx)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	if totalUsers < 300 {
		disgo.Infoln(style.Important("This repository appears to have a low amount of stargazers. Trust calculations might not be accurate."))
	}

	// For now, we only fetch contributions until 2013. It will be configurable later on
	// once the algorithm is more accurate and more data has been fetched.
	if ctx.fastMode && totalUsers > ctx.stars {
		disgo.Infof("Fetching contributions for %d users up to year %d\n", ctx.stars, 2013)
	} else {
		disgo.Infof("Fetching contributions for %d users up to year %d\n", totalUsers, 2013)
	}

	users, err := fetchContributions(ctx, cursors, 2013)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	report, err := computeTrustReport(ctx, users)
	if err != nil {
		disgo.Errorf("Unable to compute trust report %+v\n", report)
		return fmt.Errorf("unable to compute trust report: %v", err)
	}

	renderReport(ctx.details, report)

	disgo.Infof("%s Analysis successful. %d users computed.\n", style.Success(style.SymbolCheck), len(users))

	return nil
}

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/ullaakut/astronomer/pkg/context"
	"github.com/ullaakut/astronomer/pkg/gql"
	"github.com/ullaakut/astronomer/pkg/server"
	"github.com/ullaakut/astronomer/pkg/trust"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

func parseArguments() error {
	viper.SetEnvPrefix("astronomer")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.BoolP("verbose", "v", false, "Show extra logs (including comparative reports)")
	pflag.BoolP("all", "a", false, "Force astronomer to scall every stargazer of the repository (overrides --stars)")
	pflag.UintP("stars", "s", 1000, "Maxmimum amount of stars to scan, if fast mode is enabled")
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
	err := parseArguments()
	if err != nil {
		disgo.Errorln(style.Failure(style.SymbolCross, err))
		os.Exit(1)
	}

	disgo.SetTerminalOptions(disgo.WithColors(true), disgo.WithDebug(viper.GetBool("verbose")))

	repository := pflag.Arg(0)

	// Split repository into repo owner & repo name.
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

	ctx := &context.Context{
		RepoOwner:          repoInfo[0],
		RepoName:           repoInfo[1],
		GithubToken:        token,
		Stars:              viper.GetUint("stars"),
		CacheDirectoryPath: viper.GetString("cachedir"),
		ScanAll:            viper.GetBool("all"),
		Verbose:            viper.GetBool("verbose"),
	}

	if err := detectFakeStars(ctx); err != nil {
		disgo.Errorln(style.Failure(style.SymbolCross, " ", err))
		os.Exit(1)
	}
}

func detectFakeStars(ctx *context.Context) error {
	disgo.Infof("Beginning fetching process for repository %s/%s\n", ctx.RepoOwner, ctx.RepoName)

	cursors, totalUsers, err := gql.FetchStargazers(ctx)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	if totalUsers < 1000 {
		disgo.Infoln(style.Important("This repository appears to have a low amount of stargazers. Trust calculations might not be accurate."))
	}

	// For now, we only fetch contributions until 2013. It will be configurable later on
	// once the algorithm is more accurate and more data has been fetched.
	if !ctx.ScanAll && totalUsers > ctx.Stars {
		disgo.Infof("Fetching contributions for %d users up to year %d\n", ctx.Stars, 2013)
	} else {
		disgo.Infof("Fetching contributions for %d users up to year %d\n", totalUsers, 2013)
	}

	users, err := gql.FetchContributions(ctx, cursors, 2013)
	if err != nil {
		return fmt.Errorf("failed to query stargazer data: %s", err)
	}

	report, err := trust.Compute(ctx, users)
	if err != nil {
		return fmt.Errorf("unable to compute trust report: %v", err)
	}

	trust.Render(report, true)

	err = server.SendReport(report)
	if err != nil {
		return fmt.Errorf("unable to send trust report: %v", err)
	}

	disgo.Infof("%s Analysis successful. %d users computed.\n", style.Success(style.SymbolCheck), len(users))

	return nil
}

package main

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/ullaakut/disgo"
)

// "Good" average values represent what a legitimate repository should have on average.
// To reach a 99% trust level, a repository's users but have averages equivalent to at
// least 1.5 times what is considered good.
const (
	goodAverageTotalContributions       = 1600
	goodAverageTrustScore               = 10000
	goodSixtyFifthPercentileTrustScore  = 2000
	goodEightyFifthPercentileTrustScore = 15000
	goodNinetyFifthPercentileTrustScore = 26000
	goodAverageAccountAge               = 1800

	badTrustPercent  = 0.25
	goodTrustPercent = 0.75
)

type trustFactor struct {
	value        float64
	trustPercent float64
}

// trustReport represents the result of the trust computation of a repository's
// stargazers. It contains every trust factor that has been computed.
type trustReport struct {
	contributions trustFactor
	accountAge    trustFactor
	trustScore    trustFactor

	// If the amount of users is too low, those trust factors will not be computed.
	trustSFPercentile *trustFactor
	trustEFPercentile *trustFactor
	trustNFPercentile *trustFactor

	overallTrust trustFactor
}

// computeTrustReport computes all trust factors for the stargazers of a repository.
func computeTrustReport(sg []User) (*trustReport, error) {
	var (
		contribData []float64
		trustData   []float64
		ageData     []float64
		err         error
	)

	now := time.Now().Year()

	for idx := range sg {
		// Calculate the total contributions of this user, and define their score
		// by making older contributions more trustworthy.
		var totalContributions float64
		for _, contributionYear := range sg[idx].Contributions.Years {
			year, err := strconv.Atoi(contributionYear.Year)
			if err != nil {
				return nil, fmt.Errorf("invalid data format in github contributions: %v", err)
			}

			contributions := float64(contributionYear.Total)

			// How old these contributions are in years (starts at one)
			contributionAge := float64((now - year) + 1)

			// Consider contributions more trustworthy if they are older.
			sg[idx].Trustworthiness += contributions * math.Pow(contributionAge, 2)

			totalContributions += contributions
		}

		accountAge, err := sg[idx].Age()
		if err != nil {
			return nil, fmt.Errorf("unable to get account age: %v", err)
		}

		// Gather contribution, account age and trust score data for each user.
		contribData = append(contribData, totalContributions)
		ageData = append(ageData, accountAge)
		trustData = append(trustData, sg[idx].Trustworthiness)
	}

	disgo.StartStep("Computing trust levels")

	report := &trustReport{}
	err = computeContributionRatio(report, contribData)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute contribution ratio: %v", err)
	}

	err = computeWeightedContribs(report, trustData)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute weighted contributions: %v", err)
	}

	err = computeAge(report, ageData)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute account age: %v", err)
	}

	disgo.EndStep()

	err = computeOverallTrust(report)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute overall account: %v", err)
	}

	return report, nil
}

// Computes a trust % from the lifetime contributions of the stargazers. It takes into account
// the amount of average lifetime contributions.
func computeContributionRatio(report *trustReport, contribData []float64) error {
	// Compute average lifetime contributions.
	averageContributions, err := stats.Mean(contribData)
	if err != nil {
		return fmt.Errorf("unable to compute average contributions: %v", err)
	}

	totalContributionTrustPercent := computeTrustFromScore(averageContributions, goodAverageTotalContributions)

	report.contributions = trustFactor{
		value:        averageContributions,
		trustPercent: totalContributionTrustPercent,
	}

	return nil
}

// Computes a trust % from the lifetime contributions of the stargazers. These have are weighted
// so that older contributions are worth higher scores.
func computeWeightedContribs(report *trustReport, weightedContributionData []float64) error {
	averageTrust, err := stats.Mean(weightedContributionData)
	if err != nil {
		return fmt.Errorf("unable to compute average trust: %v", err)
	}

	averageScoreTrustPercent := computeTrustFromScore(averageTrust, goodAverageTrustScore)

	report.trustScore = trustFactor{
		value:        averageTrust,
		trustPercent: averageScoreTrustPercent,
	}

	// Need enough entries to be able to compute percentiles.
	if len(weightedContributionData) < 10 {
		return nil
	}

	sixtyFifthPercentile, err := stats.Percentile(weightedContributionData, 65)
	if err != nil {
		return fmt.Errorf("unable to compute 65th trust percentile: %v", err)
	}

	averageSFScoreTrustPercent := computeTrustFromScore(sixtyFifthPercentile, goodSixtyFifthPercentileTrustScore)

	report.trustSFPercentile = &trustFactor{
		value:        sixtyFifthPercentile,
		trustPercent: averageSFScoreTrustPercent,
	}

	eightyFifthPercentile, err := stats.Percentile(weightedContributionData, 85)
	if err != nil {
		return fmt.Errorf("unable to compute 85th trust percentile: %v", err)
	}

	averageEFScoreTrustPercent := computeTrustFromScore(eightyFifthPercentile, goodEightyFifthPercentileTrustScore)

	report.trustEFPercentile = &trustFactor{
		value:        eightyFifthPercentile,
		trustPercent: averageEFScoreTrustPercent,
	}

	ninetyFifthPercentile, err := stats.Percentile(weightedContributionData, 95)
	if err != nil {
		return fmt.Errorf("unable to compute 95th trust percentile: %v", err)
	}

	averageNFScoreTrustPercent := computeTrustFromScore(ninetyFifthPercentile, goodNinetyFifthPercentileTrustScore)

	report.trustNFPercentile = &trustFactor{
		value:        ninetyFifthPercentile,
		trustPercent: averageNFScoreTrustPercent,
	}

	return nil
}

// Computes a trust % from the average account age of the stargazers.
func computeAge(report *trustReport, ageData []float64) error {
	averageAccountAge, err := stats.Mean(ageData)
	if err != nil {
		return fmt.Errorf("unable to compute average account age: %v", err)
	}

	ageTrustPercent := computeTrustFromScore(averageAccountAge, goodAverageAccountAge)

	report.accountAge = trustFactor{
		value:        averageAccountAge,
		trustPercent: ageTrustPercent,
	}

	return nil
}

// Computes an overall trust % from all other trust factors.
func computeOverallTrust(report *trustReport) error {
	trustFactors := []float64{
		report.accountAge.trustPercent,

		// The trustScore and average contributions are more important so they are
		// there twice, in order to double their weight in the calculations.
		report.trustScore.trustPercent, report.trustScore.trustPercent,
		report.contributions.trustPercent, report.contributions.trustPercent,
	}

	if report.trustSFPercentile != nil && report.trustEFPercentile != nil && report.trustNFPercentile != nil {
		trustFactors = append(trustFactors, []float64{
			report.trustSFPercentile.trustPercent,
			report.trustEFPercentile.trustPercent,
			report.trustNFPercentile.trustPercent,
		}...)
	}

	overallTrustPercent, err := stats.Mean(trustFactors)
	if err != nil {
		return fmt.Errorf("unable to compute average overall trust: %v", err)
	}

	report.overallTrust = trustFactor{
		trustPercent: overallTrustPercent,
	}

	return nil
}

// computeTrustFromScore takes a score and a good expected score,
// and computes a trust level depending on the difference between
// both. Trust will reach 0.99 if the score is over 1.5 times what
// is considered a good score.
func computeTrustFromScore(score, goodScore float64) float64 {
	trust := score / (1.5 * goodScore)
	if trust > 0.99 {
		trust = 0.99
	}

	return trust
}

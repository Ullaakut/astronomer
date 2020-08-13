package trust

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Ullaakut/astronomer/pkg/context"
	"github.com/Ullaakut/astronomer/pkg/gql"
	"github.com/Ullaakut/disgo"
	"github.com/Ullaakut/disgo/style"
	"github.com/montanaflynn/stats"
)

// Factor represents one of the trust factors used to compte
// the trust score for a repository.
type Factor struct {
	// The raw value of this factor.
	Value float64

	// The % of trust given to the value, compared
	// to global references.
	TrustPercent float64
}

// FactorName is a typed string representing a
// Factor's name.
type FactorName string

// Percentile is a typed string representing
// a percentile trust factor. It is a string
// instead of a float to allow the `encoding/json`
// package to marshal trust reports.
type Percentile string

// Report represents the result of the trust computation of a repository's
// stargazers. It contains every trust factor that has been computed.
type Report struct {
	Factors     map[FactorName]Factor
	Percentiles map[Percentile]Factor
}

// Compute computes all trust factors for the stargazers of a repository.
func Compute(ctx *context.Context, users []gql.User) (*Report, error) {
	trustData := make(map[FactorName][]float64)
	now := time.Now().Year()

	for idx := range users {
		var contributionScore float64
		for year, contributions := range users[idx].YearlyContributions {
			// How old these contributions are in years (starts at one)
			contributionAge := float64((now - year) + 1)

			// Consider contributions more trustworthy if they are older.
			contributionScore += float64(contributions) * math.Pow(contributionAge, 2)
		}

		// Gather all contribution data and account ages.
		trustData[PrivateContributionFactor] = append(trustData[PrivateContributionFactor], float64(users[idx].Contributions.PrivateContributions))
		trustData[IssueContributionFactor] = append(trustData[IssueContributionFactor], float64(users[idx].Contributions.TotalIssueContributions))
		trustData[CommitContributionFactor] = append(trustData[CommitContributionFactor], float64(users[idx].Contributions.TotalCommitContributions))
		trustData[RepoContributionFactor] = append(trustData[RepoContributionFactor], float64(users[idx].Contributions.TotalRepositoryContributions))
		trustData[PRContributionFactor] = append(trustData[PRContributionFactor], float64(users[idx].Contributions.TotalPullRequestContributions))
		trustData[PRReviewContributionFactor] = append(trustData[PRReviewContributionFactor], float64(users[idx].Contributions.TotalPullRequestReviewContributions))
		trustData[AccountAgeFactor] = append(trustData[AccountAgeFactor], users[idx].DaysOld())
		trustData[ContributionScoreFactor] = append(trustData[ContributionScoreFactor], contributionScore)
	}

	disgo.StartStepf("Building trust report")

	defer disgo.EndStep()

	if uint(len(users)) > 219 {
		return buildComparativeReport(trustData)
	}

	return buildReport(trustData)
}

func buildReport(trustData map[FactorName][]float64) (*Report, error) {
	report := &Report{
		Factors: make(map[FactorName]Factor),
	}

	for factor, data := range trustData {
		score, err := stats.Mean(data)
		if err != nil {
			return nil, disgo.FailStepf("unable to compute score for factor %q: %v", factor, err)
		}

		trustPercent := computeTrustFromScore(score, factorReferences[factor])
		report.Factors[factor] = Factor{
			Value:        score,
			TrustPercent: trustPercent,
		}
	}

	// Only compute percentiles if  there are enough stargazers to be
	// able to compute every fifth percentile.
	if len(trustData[ContributionScoreFactor]) > 20 {
		report.Percentiles = make(map[Percentile]Factor)
		for _, percentile := range percentiles {
			// Error is ignored on purpose.
			pctl, _ := strconv.ParseFloat(string(percentile), 64)

			value, err := stats.Percentile(trustData[ContributionScoreFactor], pctl)
			if err != nil {
				return nil, fmt.Errorf("unable to compute score trust %sth percentile: %v", percentile, err)
			}

			report.Percentiles[percentile] = Factor{
				Value:        value,
				TrustPercent: computeTrustFromScore(value, percentileReferences[percentile]),
			}
		}
	}

	var allTrust []float64
	for factorName, weight := range factorWeights {
		for i := 0; i < weight; i++ {
			allTrust = append(allTrust, report.Factors[factorName].TrustPercent)
		}
	}

	// Take percentiles into consideration, if they were
	// computed.
	for _, percentileTrust := range report.Percentiles {
		allTrust = append(allTrust, percentileTrust.TrustPercent)
	}

	trust, err := stats.Mean(allTrust)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute overall trust: %v", err)
	}

	report.Factors[Overall] = Factor{
		TrustPercent: trust,
	}

	return report, nil
}

// buildComparativeReport splits the trust data and percentiles between the first stargazers
// and current stargazers, and it then builds a report that contains the worst of both sets.
func buildComparativeReport(trustData map[FactorName][]float64) (*Report, error) {
	report := &Report{
		Factors:     make(map[FactorName]Factor),
		Percentiles: make(map[Percentile]Factor),
	}

	firstStarsTrust, currentStarsTrust := splitTrustData(trustData)

	// Compute one trust report for the early stargazers.
	firstStarsReport, err := buildReport(firstStarsTrust)
	if err != nil {
		return nil, err
	}

	disgo.Debugln(style.Important("First 200 stargazers"))

	Render(firstStarsReport, false)

	// Compute another trust report for the random stargazers.
	currentStarsReport, err := buildReport(currentStarsTrust)
	if err != nil {
		return nil, err
	}

	disgo.Debugln(style.Important(len(currentStarsTrust[ContributionScoreFactor]), " random stargazers"))

	Render(currentStarsReport, false)

	// Build comparative report using data from both sets.
	for _, factor := range factors {
		if firstStarsReport.Factors[factor].TrustPercent <= currentStarsReport.Factors[factor].TrustPercent {
			report.Factors[factor] = firstStarsReport.Factors[factor]
		} else {
			report.Factors[factor] = currentStarsReport.Factors[factor]
		}
	}

	for _, percentile := range percentiles {
		// Skip percentiles if the random sample is too small to have percentiles.
		if currentStarsReport.Percentiles[percentile].TrustPercent == 0 {
			report.Percentiles = nil
			break
		}

		if firstStarsReport.Percentiles[percentile].TrustPercent <= currentStarsReport.Percentiles[percentile].TrustPercent {
			report.Percentiles[percentile] = firstStarsReport.Percentiles[percentile]
		} else {
			report.Percentiles[percentile] = currentStarsReport.Percentiles[percentile]
		}
	}

	var allTrust []float64
	for factorName, weight := range factorWeights {
		for i := 0; i < weight; i++ {
			allTrust = append(allTrust, report.Factors[factorName].TrustPercent)
		}
	}

	for _, percentile := range percentiles {
		// Skip percentiles if the random sample is too small to have percentiles.
		if report.Percentiles[percentile].TrustPercent == 0 {
			break
		}
		allTrust = append(allTrust, report.Percentiles[percentile].TrustPercent)
	}

	trust, err := stats.Mean(allTrust)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute overall trust: %v", err)
	}

	report.Factors[Overall] = Factor{
		TrustPercent: trust,
	}

	return report, nil
}

// splitTrustData split a trust data map between first and random stargazers.
func splitTrustData(trustData map[FactorName][]float64) (first, current map[FactorName][]float64) {
	total := len(trustData[ContributionScoreFactor])

	// Compute first stars.
	first = make(map[FactorName][]float64)
	current = make(map[FactorName][]float64)
	for _, factor := range factors {
		for i := 0; i < 200; i++ {
			first[factor] = append(first[factor], trustData[factor][i])
		}

		for i := 200; i < total; i++ {
			current[factor] = append(current[factor], trustData[factor][i])
		}
	}

	return first, current
}

// computeTrustFromScore takes a score and a reference expected score,
// and computes a trust level depending on the difference between
// both. Trust will reach 0.99 if the score is over 1.5 times what
// is considered a good score.
func computeTrustFromScore(score, reference float64) float64 {
	trust := score / (1.5 * reference)
	if trust > 0.99 {
		trust = 0.99
	}

	return trust
}

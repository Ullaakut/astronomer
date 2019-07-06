package main

import (
	"fmt"
	"math"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

type trustFactor struct {
	value        float64
	trustPercent float64
}

type factorName string

// trustReport represents the result of the trust computation of a repository's
// stargazers. It contains every trust factor that has been computed.
type trustReport struct {
	factors     map[factorName]trustFactor
	percentiles map[float64]trustFactor
}

// computeTrustReport computes all trust factors for the stargazers of a repository.
func computeTrustReport(ctx context, users []user) (*trustReport, error) {
	trustData := make(map[factorName][]float64)
	now := time.Now().Year()

	for idx := range users {
		var contributionScore float64
		for year, contributions := range users[idx].yearlyContributions {
			// How old these contributions are in years (starts at one)
			contributionAge := float64((now - year) + 1)

			// Consider contributions more trustworthy if they are older.
			contributionScore += float64(contributions) * math.Pow(contributionAge, 2)
		}

		// Gather all contribution data and account ages.
		trustData[privateContributionFactor] = append(trustData[privateContributionFactor], float64(users[idx].Contributions.PrivateContributions))
		trustData[issueContributionFactor] = append(trustData[issueContributionFactor], float64(users[idx].Contributions.TotalIssueContributions))
		trustData[commitContributionFactor] = append(trustData[commitContributionFactor], float64(users[idx].Contributions.TotalCommitContributions))
		trustData[repoContributionFactor] = append(trustData[repoContributionFactor], float64(users[idx].Contributions.TotalRepositoryContributions))
		trustData[prContributionFactor] = append(trustData[prContributionFactor], float64(users[idx].Contributions.TotalPullRequestContributions))
		trustData[prReviewContributionFactor] = append(trustData[prReviewContributionFactor], float64(users[idx].Contributions.TotalPullRequestReviewContributions))
		trustData[accountAgeFactor] = append(trustData[accountAgeFactor], users[idx].daysOld())
		trustData[contributionScoreFactor] = append(trustData[contributionScoreFactor], contributionScore)
	}

	disgo.StartStepf("Building trust report")

	defer disgo.EndStep()

	if ctx.stars == uint(len(users)) {
		return buildComparativeReport(trustData)
	}

	return buildReport(trustData)
}

func buildReport(trustData map[factorName][]float64) (*trustReport, error) {
	report := &trustReport{
		factors:     make(map[factorName]trustFactor),
		percentiles: make(map[float64]trustFactor),
	}

	for factor, data := range trustData {
		score, err := stats.Mean(data)
		if err != nil {
			return nil, disgo.FailStepf("unable to compute score for factor %q: %v", factor, err)
		}

		trustPercent := computeTrustFromScore(score, factorReferences[factor])
		report.factors[factor] = trustFactor{
			value:        score,
			trustPercent: trustPercent,
		}
	}

	// Only compute percentiles if  there are enough stargazers to be
	// able to compute every fifth percentile.
	if len(trustData[contributionScoreFactor]) > 20 {
		for _, percentile := range percentiles {
			value, err := stats.Percentile(trustData[contributionScoreFactor], percentile)
			if err != nil {
				return nil, fmt.Errorf("unable to compute score trust %1.fth percentile: %v", percentile, err)
			}

			report.percentiles[percentile] = trustFactor{
				value:        value,
				trustPercent: computeTrustFromScore(value, percentileReferences[percentile]),
			}
		}
	}

	var allTrust []float64
	for factorName, weight := range factorWeights {
		for i := 0; i < weight; i++ {
			allTrust = append(allTrust, report.factors[factorName].trustPercent)
		}
	}

	// Take percentiles into consideration, if they were
	// computed.
	for _, percentileTrust := range report.percentiles {
		allTrust = append(allTrust, percentileTrust.trustPercent)
	}

	trust, err := stats.Mean(allTrust)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute overall trust: %v", err)
	}

	report.factors[overallTrust] = trustFactor{
		trustPercent: trust,
	}

	return report, nil
}

// buildComparativeReport splits the trust data and percentiles between the first stargazers
// and current stargazers, and it then builds a report that contains the worst of both sets.
func buildComparativeReport(trustData map[factorName][]float64) (*trustReport, error) {
	report := &trustReport{
		factors:     make(map[factorName]trustFactor),
		percentiles: make(map[float64]trustFactor),
	}

	firstStarsTrust, currentStarsTrust := splitTrustData(trustData)

	// Compute one trust report for the early stargazers.
	firstStarsReport, err := buildReport(firstStarsTrust)
	if err != nil {
		return nil, err
	}

	disgo.Debugln(style.Important("First 200 stargazers"))

	renderReport(firstStarsReport, true)

	// Compute another trust report for the random stargazers.
	currentStarsReport, err := buildReport(currentStarsTrust)
	if err != nil {
		return nil, err
	}

	disgo.Debugln(style.Important(len(currentStarsTrust[contributionScoreFactor]), " random stargazers"))

	renderReport(currentStarsReport, true)

	// Build comparative report using data from both sets.
	for _, factor := range factors {
		if firstStarsReport.factors[factor].trustPercent <= currentStarsReport.factors[factor].trustPercent {
			report.factors[factor] = firstStarsReport.factors[factor]
		} else {
			report.factors[factor] = currentStarsReport.factors[factor]
		}
	}

	for _, percentile := range percentiles {
		if firstStarsReport.percentiles[percentile].trustPercent <= currentStarsReport.percentiles[percentile].trustPercent {
			report.percentiles[percentile] = firstStarsReport.percentiles[percentile]
		} else {
			report.percentiles[percentile] = currentStarsReport.percentiles[percentile]
		}
	}

	var allTrust []float64
	for factorName, weight := range factorWeights {
		for i := 0; i < weight; i++ {
			allTrust = append(allTrust, report.factors[factorName].trustPercent)
		}
	}

	for _, percentile := range percentiles {
		allTrust = append(allTrust, report.percentiles[percentile].trustPercent)
	}

	trust, err := stats.Mean(allTrust)
	if err != nil {
		return nil, disgo.FailStepf("unable to compute overall trust: %v", err)
	}

	report.factors[overallTrust] = trustFactor{
		trustPercent: trust,
	}

	return report, nil
}

// splitTrustData split a trust data map between first and random stargazers.
func splitTrustData(trustData map[factorName][]float64) (first, current map[factorName][]float64) {
	total := len(trustData[contributionScoreFactor])

	// Compute first stars.
	first = make(map[factorName][]float64)
	current = make(map[factorName][]float64)
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

package main

import (
	"fmt"
	"math"
	"time"

	"github.com/montanaflynn/stats"
)

type trustFactor struct {
	value        float64
	trustPercent float64
}

type factorName string

// trustReport represents the result of the trust computation of a repository's
// stargazers. It contains every trust factor that has been computed.
type trustReport map[factorName]trustFactor

const (
	privateContributionFactor  factorName = "Private contributions"
	contributionScoreFactor               = "Weighted contributions"
	issueContributionFactor               = "Created issues"
	commitContributionFactor              = "Commits authored"
	repoContributionFactor                = "Repositories"
	prContributionFactor                  = "Pull requests"
	prReviewContributionFactor            = "Code reviews"
	accountAgeFactor                      = "Account age (days)"
	overallTrust                          = "Overall trust"
)

var (
	factorReferences = map[factorName]float64{
		privateContributionFactor:  250,
		contributionScoreFactor:    15000,
		issueContributionFactor:    30,
		commitContributionFactor:   300,
		repoContributionFactor:     30,
		prContributionFactor:       20,
		prReviewContributionFactor: 15,
		accountAgeFactor:           1600,
	}

	// factorWeights represents the importance of each factor in
	// the calculation of the overall trust factor.
	factorWeights = map[factorName]int{
		privateContributionFactor:  1,
		issueContributionFactor:    3,
		commitContributionFactor:   3,
		repoContributionFactor:     2,
		prContributionFactor:       2,
		prReviewContributionFactor: 2,
		contributionScoreFactor:    8,
		accountAgeFactor:           2,
	}

	factors = []factorName{
		contributionScoreFactor,
		privateContributionFactor,
		issueContributionFactor,
		commitContributionFactor,
		repoContributionFactor,
		prContributionFactor,
		prReviewContributionFactor,
		accountAgeFactor,
	}
)

// computeTrustReport computes all trust factors for the stargazers of a repository.
func computeTrustReport(users []user) (trustReport, error) {
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

	return buildReport(trustData)
}

func buildReport(trustData map[factorName][]float64) (trustReport, error) {
	report := trustReport{}

	for factor, data := range trustData {
		score, err := stats.Mean(data)
		if err != nil {
			return nil, fmt.Errorf("unable to compute score for factor %q: %v", factor, err)
		}

		trustPercent := computeTrustFromScore(score, factorReferences[factor])
		report[factor] = trustFactor{
			value:        score,
			trustPercent: trustPercent,
		}
	}

	var allTrust []float64
	for factorName, weight := range factorWeights {
		for i := 0; i < weight; i++ {
			allTrust = append(allTrust, report[factorName].trustPercent)
		}
	}

	trust, err := stats.Mean(allTrust)
	if err != nil {
		return nil, fmt.Errorf("unable to compute overall trust: %v", err)
	}

	report[overallTrust] = trustFactor{
		trustPercent: trust,
	}

	return report, nil
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

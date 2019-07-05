package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReport(t *testing.T) {
	expectedFactors := map[factorName]trustFactor{
		privateContributionFactor:  trustFactor{value: 2 * factorReferences[privateContributionFactor], trustPercent: 0.99},
		issueContributionFactor:    trustFactor{value: 2 * factorReferences[issueContributionFactor], trustPercent: 0.99},
		commitContributionFactor:   trustFactor{value: 2 * factorReferences[commitContributionFactor], trustPercent: 0.99},
		repoContributionFactor:     trustFactor{value: 2 * factorReferences[repoContributionFactor], trustPercent: 0.99},
		prContributionFactor:       trustFactor{value: 2 * factorReferences[prContributionFactor], trustPercent: 0.99},
		prReviewContributionFactor: trustFactor{value: 2 * factorReferences[prReviewContributionFactor], trustPercent: 0.99},
		accountAgeFactor:           trustFactor{value: 2 * factorReferences[accountAgeFactor], trustPercent: 0.99},
		contributionScoreFactor:    trustFactor{value: 2 * factorReferences[contributionScoreFactor], trustPercent: 0.99},
	}

	trustData := map[factorName][]float64{
		privateContributionFactor:  []float64{0, 4 * factorReferences[privateContributionFactor], 2 * factorReferences[privateContributionFactor]},
		issueContributionFactor:    []float64{0, 2 * factorReferences[issueContributionFactor], 4 * factorReferences[issueContributionFactor]},
		commitContributionFactor:   []float64{0, 2 * factorReferences[commitContributionFactor], 4 * factorReferences[commitContributionFactor]},
		repoContributionFactor:     []float64{0, 2 * factorReferences[repoContributionFactor], 4 * factorReferences[repoContributionFactor]},
		prContributionFactor:       []float64{0, 2 * factorReferences[prContributionFactor], 4 * factorReferences[prContributionFactor]},
		prReviewContributionFactor: []float64{0, 2 * factorReferences[prReviewContributionFactor], 4 * factorReferences[prReviewContributionFactor]},
		accountAgeFactor:           []float64{0, 2 * factorReferences[accountAgeFactor], 4 * factorReferences[accountAgeFactor]},
		contributionScoreFactor:    []float64{0, 2 * factorReferences[contributionScoreFactor], 4 * factorReferences[contributionScoreFactor]},
	}

	report, err := buildReport(trustData)
	require.NoError(t, err)
	require.NotNil(t, report)

	for factor, expectedTrust := range expectedFactors {
		assert.Equal(t, expectedTrust, report.factors[factor], "unexpected value for factor %q", factor)
	}
}

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

	percentiles := map[float64]trustFactor{
		5:  trustFactor{value: percentileReferences[5], trustPercent: 0.75},
		10: trustFactor{value: percentileReferences[10], trustPercent: 0.75},
		15: trustFactor{value: percentileReferences[15], trustPercent: 0.75},
		20: trustFactor{value: percentileReferences[20], trustPercent: 0.75},
		25: trustFactor{value: percentileReferences[25], trustPercent: 0.75},
		30: trustFactor{value: percentileReferences[30], trustPercent: 0.75},
		35: trustFactor{value: percentileReferences[35], trustPercent: 0.75},
		40: trustFactor{value: percentileReferences[40], trustPercent: 0.75},
		45: trustFactor{value: percentileReferences[45], trustPercent: 0.75},
		50: trustFactor{value: percentileReferences[50], trustPercent: 0.75},
		55: trustFactor{value: percentileReferences[55], trustPercent: 0.75},
		60: trustFactor{value: percentileReferences[60], trustPercent: 0.75},
		65: trustFactor{value: percentileReferences[65], trustPercent: 0.75},
		70: trustFactor{value: percentileReferences[70], trustPercent: 0.75},
		75: trustFactor{value: percentileReferences[75], trustPercent: 0.75},
		80: trustFactor{value: percentileReferences[80], trustPercent: 0.75},
		85: trustFactor{value: percentileReferences[85], trustPercent: 0.75},
		90: trustFactor{value: percentileReferences[90], trustPercent: 0.75},
		95: trustFactor{value: percentileReferences[95], trustPercent: 0.75},
	}

	report, err := buildReport(trustData, percentiles)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, percentiles, report.percentiles)
	for factor, expectedTrust := range expectedFactors {
		assert.Equal(t, expectedTrust, report.factors[factor], "unexpected value for factor %q", factor)
	}
}

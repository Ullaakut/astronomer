package trust

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReport(t *testing.T) {
	expectedFactors := map[FactorName]Factor{
		PrivateContributionFactor:  Factor{Value: 2 * factorReferences[PrivateContributionFactor], TrustPercent: 0.99},
		IssueContributionFactor:    Factor{Value: 2 * factorReferences[IssueContributionFactor], TrustPercent: 0.99},
		CommitContributionFactor:   Factor{Value: 2 * factorReferences[CommitContributionFactor], TrustPercent: 0.99},
		RepoContributionFactor:     Factor{Value: 2 * factorReferences[RepoContributionFactor], TrustPercent: 0.99},
		PRContributionFactor:       Factor{Value: 2 * factorReferences[PRContributionFactor], TrustPercent: 0.99},
		PRReviewContributionFactor: Factor{Value: 2 * factorReferences[PRReviewContributionFactor], TrustPercent: 0.99},
		AccountAgeFactor:           Factor{Value: 2 * factorReferences[AccountAgeFactor], TrustPercent: 0.99},
		ContributionScoreFactor:    Factor{Value: 2 * factorReferences[ContributionScoreFactor], TrustPercent: 0.99},
	}

	trustData := map[FactorName][]float64{
		PrivateContributionFactor:  []float64{0, 4 * factorReferences[PrivateContributionFactor], 2 * factorReferences[PrivateContributionFactor]},
		IssueContributionFactor:    []float64{0, 2 * factorReferences[IssueContributionFactor], 4 * factorReferences[IssueContributionFactor]},
		CommitContributionFactor:   []float64{0, 2 * factorReferences[CommitContributionFactor], 4 * factorReferences[CommitContributionFactor]},
		RepoContributionFactor:     []float64{0, 2 * factorReferences[RepoContributionFactor], 4 * factorReferences[RepoContributionFactor]},
		PRContributionFactor:       []float64{0, 2 * factorReferences[PRContributionFactor], 4 * factorReferences[PRContributionFactor]},
		PRReviewContributionFactor: []float64{0, 2 * factorReferences[PRReviewContributionFactor], 4 * factorReferences[PRReviewContributionFactor]},
		AccountAgeFactor:           []float64{0, 2 * factorReferences[AccountAgeFactor], 4 * factorReferences[AccountAgeFactor]},
		ContributionScoreFactor:    []float64{0, 2 * factorReferences[ContributionScoreFactor], 4 * factorReferences[ContributionScoreFactor]},
	}

	report, err := buildReport(trustData)
	require.NoError(t, err)
	require.NotNil(t, report)

	for factor, expectedTrust := range expectedFactors {
		assert.Equal(t, expectedTrust, report.Factors[factor], "unexpected value for factor %q", factor)
	}
}

func TestBuildReportWithPercentiles(t *testing.T) {
	expectedFactors := map[FactorName]Factor{
		PrivateContributionFactor:  Factor{Value: 0, TrustPercent: 0},
		IssueContributionFactor:    Factor{Value: 0, TrustPercent: 0},
		CommitContributionFactor:   Factor{Value: 0, TrustPercent: 0},
		RepoContributionFactor:     Factor{Value: 0, TrustPercent: 0},
		PRContributionFactor:       Factor{Value: 0, TrustPercent: 0},
		PRReviewContributionFactor: Factor{Value: 0, TrustPercent: 0},
		AccountAgeFactor:           Factor{Value: 0, TrustPercent: 0},
		ContributionScoreFactor:    Factor{Value: 0, TrustPercent: 0},
	}

	expectedPercentiles := map[Percentile]Factor{
		percentiles[0]:  Factor{TrustPercent: 0},
		percentiles[1]:  Factor{TrustPercent: 0},
		percentiles[2]:  Factor{TrustPercent: 0},
		percentiles[3]:  Factor{TrustPercent: 0},
		percentiles[4]:  Factor{TrustPercent: 0},
		percentiles[5]:  Factor{TrustPercent: 0},
		percentiles[6]:  Factor{TrustPercent: 0},
		percentiles[7]:  Factor{TrustPercent: 0},
		percentiles[8]:  Factor{TrustPercent: 0},
		percentiles[9]:  Factor{TrustPercent: 0},
		percentiles[10]: Factor{TrustPercent: 0},
		percentiles[11]: Factor{TrustPercent: 0},
		percentiles[12]: Factor{TrustPercent: 0},
		percentiles[13]: Factor{TrustPercent: 0},
		percentiles[14]: Factor{TrustPercent: 0},
		percentiles[15]: Factor{TrustPercent: 0},
		percentiles[16]: Factor{TrustPercent: 0},
		percentiles[17]: Factor{TrustPercent: 0},
		percentiles[18]: Factor{TrustPercent: 0},
	}

	trustData := map[FactorName][]float64{
		PrivateContributionFactor:  []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		IssueContributionFactor:    []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		CommitContributionFactor:   []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		RepoContributionFactor:     []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		PRContributionFactor:       []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		PRReviewContributionFactor: []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		AccountAgeFactor:           []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ContributionScoreFactor:    []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	report, err := buildReport(trustData)
	require.NoError(t, err)
	require.NotNil(t, report)

	for factor, expectedTrust := range expectedFactors {
		assert.Equal(t, expectedTrust, report.Factors[factor], "unexpected value for factor %q", factor)
	}

	for percentile, expectedPercentile := range expectedPercentiles {
		assert.Equal(t, expectedPercentile, report.Percentiles[percentile], "unexpected value for percentile %q", percentile)
	}
}

func TestSplitTrustReports(t *testing.T) {
	trustData := make(map[FactorName][]float64)

	earlyUser := float64(1)
	randomUser := float64(2)

	trustData = addToTrustData(trustData, 200, earlyUser)
	trustData = addToTrustData(trustData, 800, randomUser)

	earlyUsers, randomUsers := splitTrustData(trustData)
	require.NotNil(t, earlyUsers)
	require.NotNil(t, randomUsers)

	for _, data := range earlyUsers {
		assert.Len(t, data, 200)
		for _, userValue := range data {
			assert.Equal(t, userValue, earlyUser)
		}
	}

	for _, data := range randomUsers {
		assert.Len(t, data, 800)
		for _, userValue := range data {
			assert.Equal(t, userValue, randomUser)
		}
	}
}

func addToTrustData(trustData map[FactorName][]float64, amount int, value float64) map[FactorName][]float64 {
	for i := 0; i < amount; i++ {
		for _, factor := range factors {
			trustData[factor] = append(trustData[factor], value)
		}
	}

	return trustData
}

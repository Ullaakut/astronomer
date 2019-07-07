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

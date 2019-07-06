package trust

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ullaakut/astronomer/pkg/context"
)

func TestBuildReport(t *testing.T) {
	expectedFactors := map[FactorName]Factor{
		privateContributionFactor:  Factor{Value: 2 * factorReferences[privateContributionFactor], TrustPercent: 0.99},
		issueContributionFactor:    Factor{Value: 2 * factorReferences[issueContributionFactor], TrustPercent: 0.99},
		commitContributionFactor:   Factor{Value: 2 * factorReferences[commitContributionFactor], TrustPercent: 0.99},
		repoContributionFactor:     Factor{Value: 2 * factorReferences[repoContributionFactor], TrustPercent: 0.99},
		prContributionFactor:       Factor{Value: 2 * factorReferences[prContributionFactor], TrustPercent: 0.99},
		prReviewContributionFactor: Factor{Value: 2 * factorReferences[prReviewContributionFactor], TrustPercent: 0.99},
		accountAgeFactor:           Factor{Value: 2 * factorReferences[accountAgeFactor], TrustPercent: 0.99},
		contributionScoreFactor:    Factor{Value: 2 * factorReferences[contributionScoreFactor], TrustPercent: 0.99},
	}

	trustData := map[FactorName][]float64{
		privateContributionFactor:  []float64{0, 4 * factorReferences[privateContributionFactor], 2 * factorReferences[privateContributionFactor]},
		issueContributionFactor:    []float64{0, 2 * factorReferences[issueContributionFactor], 4 * factorReferences[issueContributionFactor]},
		commitContributionFactor:   []float64{0, 2 * factorReferences[commitContributionFactor], 4 * factorReferences[commitContributionFactor]},
		repoContributionFactor:     []float64{0, 2 * factorReferences[repoContributionFactor], 4 * factorReferences[repoContributionFactor]},
		prContributionFactor:       []float64{0, 2 * factorReferences[prContributionFactor], 4 * factorReferences[prContributionFactor]},
		prReviewContributionFactor: []float64{0, 2 * factorReferences[prReviewContributionFactor], 4 * factorReferences[prReviewContributionFactor]},
		accountAgeFactor:           []float64{0, 2 * factorReferences[accountAgeFactor], 4 * factorReferences[accountAgeFactor]},
		contributionScoreFactor:    []float64{0, 2 * factorReferences[contributionScoreFactor], 4 * factorReferences[contributionScoreFactor]},
	}

	ctx := &context.Context{
		Verbose: true,
	}

	report, err := buildReport(ctx, trustData)
	require.NoError(t, err)
	require.NotNil(t, report)

	for factor, expectedTrust := range expectedFactors {
		assert.Equal(t, expectedTrust, report.Factors[factor], "unexpected value for factor %q", factor)
	}
}

package main

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ullaakut/disgo"
)

const year = 24 * time.Hour * 365

var (
	twoYearsAgo = time.Now().Add(-2 * year).Format(time.RFC3339)

	badUser = User{
		CreatedAt: twoYearsAgo,
		Contributions: Contributions{
			Years: []YearlyContribution{
				{
					Year:  fmt.Sprint(time.Now().Year()),
					Total: 600,
				},
				{
					Year:  fmt.Sprint(time.Now().Year() - 1),
					Total: 400,
				},
			},
		},
	}

	trustReportNoPercentile = &trustReport{
		contributions: trustFactor{
			value:        1000,
			trustPercent: float64(1000) / float64(1.5*goodAverageTotalContributions),
		},
		accountAge: trustFactor{
			value:        730,
			trustPercent: float64(730) / float64(1.5*goodAverageAccountAge),
		},
		trustScore: trustFactor{
			value:        2200,
			trustPercent: float64(2200) / float64(1.5*goodAverageTrustScore),
		},
		trustSFPercentile: nil,
		trustEFPercentile: nil,
		trustNFPercentile: nil,
		overallTrust: trustFactor{
			trustPercent: ((float64(1000)/float64(1.5*goodAverageTotalContributions))*2 +
				(float64(730) / float64(1.5*goodAverageAccountAge)) +
				(float64(2200)/float64(1.5*goodAverageTrustScore))*2) /
				5,
		},
	}

	trustReportWithPercentiles = &trustReport{
		contributions: trustFactor{
			value:        1000,
			trustPercent: float64(1000) / float64(1.5*goodAverageTotalContributions), // 0.41667
		},
		accountAge: trustFactor{
			value:        730,
			trustPercent: float64(730) / float64(1.5*goodAverageAccountAge),
		},
		trustScore: trustFactor{
			value:        2200,
			trustPercent: float64(2200) / float64(1.5*goodAverageTrustScore),
		},
		trustSFPercentile: &trustFactor{
			value:        2200,
			trustPercent: float64(2200) / float64(1.5*goodSixtyFifthPercentileTrustScore),
		},
		trustEFPercentile: &trustFactor{
			value:        2200,
			trustPercent: float64(2200) / float64(1.5*goodEightyFifthPercentileTrustScore),
		},
		trustNFPercentile: &trustFactor{
			value:        2200,
			trustPercent: float64(2200) / float64(1.5*goodNinetyFifthPercentileTrustScore),
		},
		overallTrust: trustFactor{
			trustPercent: ((float64(1000)/float64(1.5*goodAverageTotalContributions))*2 +
				float64(730)/float64(1.5*goodAverageAccountAge) +
				(float64(2200)/float64(1.5*goodAverageTrustScore))*2 +
				float64(2200)/float64(1.5*goodSixtyFifthPercentileTrustScore) +
				float64(2200)/float64(1.5*goodEightyFifthPercentileTrustScore) +
				float64(2200)/float64(1.5*goodNinetyFifthPercentileTrustScore)) /
				8,
		},
	}
)

// Category                             Score           Trust
// --------                             -----           -----
// Users with >1 contribtuion:          0                 99%
// Average total contributions:         1246              52%
// Average score:                       12778             68%
// Average account age (days):          2242              83%
// ----------------------------------------------------------
// Overall trust:                                         76%
// ✔ Analysis successful. 14 users computed.

func TestComputeTrustReport(t *testing.T) {
	tests := map[string]struct {
		users []User

		expectedReport *trustReport
		expectedErr    error
	}{
		"not enough users for percentiles": {
			users: []User{
				badUser,
			},

			expectedReport: trustReportNoPercentile,
		},
		"multiple identical users": {
			users: []User{
				badUser, badUser, badUser, badUser, badUser,
				badUser, badUser, badUser, badUser, badUser,
			},

			expectedReport: trustReportWithPercentiles,
		},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			logger := &bytes.Buffer{}
			disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger))

			report, err := computeTrustReport(test.users)

			assert.Equal(t, test.expectedErr, err)
			assert.Equal(t, test.expectedReport, report)
		})
	}
}

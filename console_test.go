package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ullaakut/disgo"
)

func TestRenderReport(t *testing.T) {
	tests := map[string]struct {
		report *trustReport

		expectedRenderedLines []string
	}{
		"report without percentiles": {
			report: trustReportNoPercentile,

			expectedRenderedLines: []string{
				"Category                             Score           Trust",
				"--------                             -----           -----",
				"Average total contributions:         1000              42%",
				"Average score:                       2200              15%",
				"Average account age (days):          730               27%",
				"----------------------------------------------------------",
				"Overall trust:                                         28%",
			},
		},
		"report with percentiles": {
			report: trustReportWithPercentiles,

			expectedRenderedLines: []string{
				"Category                             Score           Trust",
				"--------                             -----           -----",
				"Average total contributions:         1000              42%",
				"Average score:                       2200              15%",
				"65th percentile:                     2200              73%",
				"85th percentile:                     2200              10%",
				"95th percentile:                     2200               6%",
				"Average account age (days):          730               27%",
				"----------------------------------------------------------",
				"Overall trust:                                         29%",
			},
		},
		"missing report": {
			report: nil,

			expectedRenderedLines: []string{
				"✖ No report to render.",
			},
		},
	}

	for description, test := range tests {
		t.Run(description, func(t *testing.T) {
			logger := &bytes.Buffer{}
			disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

			renderReport(test.report)

			for _, expectedRenderedLine := range test.expectedRenderedLines {
				assert.Contains(t, logger.String(), expectedRenderedLine)
			}
		})
	}
}

func TestPrintTrustFactor(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printTrustFactor("test_name", trustFactor{
		value:        42,
		trustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "test_name:                           42                99%")
}

func TestPrintResult(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printResult("test_name", trustFactor{
		trustPercent: 0.87,
	})

	assert.Contains(t, logger.String(), "----------------------------------------------------------")
	assert.Contains(t, logger.String(), "test_name:                                             87%")
}

func TestPrintHeader(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printHeader()

	assert.Contains(t, logger.String(), "Category                             Score           Trust")
	assert.Contains(t, logger.String(), "--------                             -----           -----")
}

package trust

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/Ullaakut/disgo"
)

func TestPrintTrustFactor(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printFactor(true, "test_name", Factor{
		Value:        42,
		TrustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "test_name:                           42                A")
}

func TestPrintPercentile(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printPercentile(true, percentiles[0], Factor{
		Value:        8484,
		TrustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "5th percentile:                      8484              A")
}

func TestPrintResult(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printResult(true, "test_name", Factor{
		TrustPercent: 0.87,
	})

	assert.Contains(t, logger.String(), "----------------------------------------------------------")
	assert.Contains(t, logger.String(), "test_name:                                             A")
}

func TestPrintHeader(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printHeader(true)

	assert.Contains(t, logger.String(), "Averages                             Score           Trust")
	assert.Contains(t, logger.String(), "--------                             -----           -----")
}

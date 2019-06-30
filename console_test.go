package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ullaakut/disgo"
)

func TestPrintTrustFactor(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printTrustFactor("test_name", trustFactor{
		value:        42,
		trustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "test_name:                           42                99%")
}

func TestPrintPercentile(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	printPercentile(42, trustFactor{
		value:        8484,
		trustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "42th percentile:                     8484              99%")
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

	assert.Contains(t, logger.String(), "Averages                             Score           Trust")
	assert.Contains(t, logger.String(), "--------                             -----           -----")
}

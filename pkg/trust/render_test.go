package trust

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ullaakut/astronomer/pkg/context"
	"github.com/ullaakut/disgo"
)

func TestPrintTrustFactor(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	ctx := &context.Context{
		Verbose: false,
	}

	printFactor(ctx, "test_name", Factor{
		Value:        42,
		TrustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "test_name:                           42                99%")
}

func TestPrintPercentile(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	ctx := &context.Context{
		Verbose: false,
	}

	printPercentile(ctx, 42, Factor{
		Value:        8484,
		TrustPercent: 0.99,
	})

	assert.Contains(t, logger.String(), "42th percentile:                     8484              99%")
}

func TestPrintResult(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	ctx := &context.Context{
		Verbose: false,
	}

	printResult(ctx, "test_name", Factor{
		TrustPercent: 0.87,
	})

	assert.Contains(t, logger.String(), "----------------------------------------------------------")
	assert.Contains(t, logger.String(), "test_name:                                             87%")
}

func TestPrintHeader(t *testing.T) {
	logger := &bytes.Buffer{}
	disgo.SetTerminalOptions(disgo.WithColors(false), disgo.WithDefaultOutput(logger), disgo.WithErrorOutput(logger))

	ctx := &context.Context{
		Verbose: false,
	}

	printHeader(ctx)

	assert.Contains(t, logger.String(), "Averages                             Score           Trust")
	assert.Contains(t, logger.String(), "--------                             -----           -----")
}

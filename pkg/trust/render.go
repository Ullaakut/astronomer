package trust

import (
	"fmt"
	"strings"

	"github.com/ullaakut/disgo"
	"github.com/ullaakut/disgo/style"
)

const (
	// Averages                             Score           Trust
	// --------                             -----           -----
	headerFormat = "\n%s<TAB>%s<TAB>%s\n%s<TAB>%s<TAB>%s\n"

	// Average score:                       12778            68%
	factorsFormat = "%s:<TAB>%s<TAB>%s\n"

	// > Overall trust:                                      76%
	overallTrustFormat = "%s\n%s:<TAB>%s\n"

	// Length of the `Averages` column.
	firstColumnLength = 35

	// Length of the `Score` column.
	secondColumnLength = 15
)

func printf(info bool, format string, s ...interface{}) {
	if info {
		disgo.Infof(format, s...)
	} else {
		disgo.Debugf(format, s...)
	}
}

// Render prints a report.
func Render(report *Report, info bool) {
	if report == nil {
		disgo.Errorln(style.Failure(style.SymbolCross, " No report to render."))
		return
	}

	printHeader(info)

	for _, factorName := range factors {
		printFactor(info, string(factorName), report.Factors[factorName])
	}

	if report.Percentiles != nil {
		for _, percentile := range percentiles {
			printPercentile(info, percentile, report.Percentiles[percentile])
		}
	}

	printResult(info, "Overall trust", report.Factors[overallTrust])
}

// printHeader prints the header containing each category name and underlines them.
func printHeader(info bool) {
	headerNames := []string{
		"Averages",
		"Score",
		"Trust",
	}

	var underlines []string
	for _, headerName := range headerNames {
		underlines = append(underlines, generateUnderlineFromHeader(headerName))
	}

	// Tabulate headers properly depending on column lengths.
	format := tabulateFormat(headerFormat, headerNames[0], firstColumnLength+1)
	format = tabulateFormat(format, headerNames[1], secondColumnLength)
	format = tabulateFormat(format, underlines[0], firstColumnLength+1)
	format = tabulateFormat(format, underlines[1], secondColumnLength)

	// Render the header.
	printf(info,
		format,
		style.Important(headerNames[0]), style.Important(headerNames[1]), style.Important(headerNames[2]),
		underlines[0], underlines[1], underlines[2],
	)
}

// printFactor prints a factor in the following format:
// FactorName:                  Score             Trust%
func printFactor(info bool, factorName string, factor Factor) {
	format := tabulateFormat(factorsFormat, factorName, firstColumnLength)
	format = tabulateFormat(format, fmt.Sprintf("%1.f", factor.Value), secondColumnLength)

	if factor.TrustPercent < 0.5 {
		printf(info, format, factorName, style.Failure(fmt.Sprintf("%1.f", factor.Value)), style.Failure(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	} else if factor.TrustPercent < 0.75 {
		printf(info, format, factorName, style.Important(fmt.Sprintf("%1.f", factor.Value)), style.Important(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	} else {
		printf(info, format, factorName, style.Success(fmt.Sprintf("%1.f", factor.Value)), style.Success(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	}
}

// printPercentile prints a percentile value in the following format:
// xth percentile:                  Score             Trust%
func printPercentile(info bool, percentile Percentile, factor Factor) {
	factorName := fmt.Sprintf("%sth percentile", percentile)

	printFactor(info, factorName, factor)
}

// printResult prints the overall result in the following format:
// FactorName:                                    Trust%
func printResult(info bool, factorName string, factor Factor) {
	format := tabulateFormat(overallTrustFormat, factorName, firstColumnLength+secondColumnLength+1)
	underline := generateUnderline(firstColumnLength + secondColumnLength + 8)

	if factor.TrustPercent < 0.5 {
		printf(info, format, underline, factorName, style.Failure(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	} else if factor.TrustPercent < 0.75 {
		printf(info, format, underline, factorName, style.Important(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	} else {
		printf(info, format, underline, factorName, style.Success(fmt.Sprintf("%4.f%%", factor.TrustPercent*100)))
	}
}

// tabulateFormat inserts spaces in formatting strings depending on variable name lengths.
func tabulateFormat(formatString, variableName string, columnLength int) string {
	var spaces string
	for i := len(variableName); i <= columnLength; i++ {
		spaces = fmt.Sprint(spaces, " ")
	}

	return strings.Replace(formatString, "<TAB>", spaces, 1)
}

// generateUnderlineFromHeader generates a string of dashes of equal
// length to the header name it's given, in order to underline it.
func generateUnderlineFromHeader(headerName string) string {
	return generateUnderline(len(headerName))
}

// generateUnderline generates a string of dashes of a given length.
func generateUnderline(length int) string {
	var underline []rune
	for i := 0; i < length; i++ {
		underline = append(underline, '-')
	}

	return string(underline)
}

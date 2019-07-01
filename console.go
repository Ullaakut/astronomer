package main

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
	trustFactorsFormat = "%s:<TAB>%s<TAB>%s\n"

	// > Overall trust:                                      76%
	overallTrustFormat = "%s\n%s:<TAB>%s\n"

	// Length of the `Averages` column.
	firstColumnLength = 35

	// Length of the `Score` column.
	secondColumnLength = 15
)

// renderReport prints a
func renderReport(details bool, report *trustReport) {
	if report == nil {
		disgo.Errorln(style.Failure(style.SymbolCross, " No report to render."))
		return
	}

	printHeader()

	for _, factorName := range factors {
		printTrustFactor(string(factorName), report.factors[factorName])
	}

	if report.percentiles != nil {
		for _, percentile := range percentiles {
			printPercentile(percentile, report.percentiles[percentile])
		}
	} else if details {
		disgo.Errorln(style.Failure(style.SymbolCross), "Not enough stargazers to be able to compute percentiles.")
	}

	printResult("Overall trust", report.factors[overallTrust])
}

// printHeader prints the header containing each category name and underlines them.
func printHeader() {
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
	disgo.Infof(
		format,
		style.Important(headerNames[0]), style.Important(headerNames[1]), style.Important(headerNames[2]),
		underlines[0], underlines[1], underlines[2],
	)
}

// printTrustFactor prints a trust factor in the following format:
// FactorName:                  Score             Trust%
func printTrustFactor(trustFactorName string, trustFactor trustFactor) {
	format := tabulateFormat(trustFactorsFormat, trustFactorName, firstColumnLength)
	format = tabulateFormat(format, fmt.Sprintf("%1.f", trustFactor.value), secondColumnLength)

	if trustFactor.trustPercent < 0.5 {
		disgo.Infof(format, trustFactorName, style.Failure(fmt.Sprintf("%1.f", trustFactor.value)), style.Failure(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else if trustFactor.trustPercent < 0.75 {
		disgo.Infof(format, trustFactorName, style.Important(fmt.Sprintf("%1.f", trustFactor.value)), style.Important(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else {
		disgo.Infof(format, trustFactorName, style.Success(fmt.Sprintf("%1.f", trustFactor.value)), style.Success(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	}
}

// printPercentile prints a percentile value in the following format:
// xth percentile:                  Score             Trust%
func printPercentile(percentile float64, trustFactor trustFactor) {
	trustFactorName := fmt.Sprintf("%2.fth percentile", percentile)

	format := tabulateFormat(trustFactorsFormat, trustFactorName, firstColumnLength)
	format = tabulateFormat(format, fmt.Sprintf("%1.f", trustFactor.value), secondColumnLength)

	if trustFactor.trustPercent < 0.5 {
		disgo.Infof(format, trustFactorName, style.Failure(fmt.Sprintf("%1.f", trustFactor.value)), style.Failure(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else if trustFactor.trustPercent < 0.75 {
		disgo.Infof(format, trustFactorName, style.Important(fmt.Sprintf("%1.f", trustFactor.value)), style.Important(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else {
		disgo.Infof(format, trustFactorName, style.Success(fmt.Sprintf("%1.f", trustFactor.value)), style.Success(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	}
}

// printResult prints the overall result in the following format:
// FactorName:                                    Trust%
func printResult(trustFactorName string, trustFactor trustFactor) {
	format := tabulateFormat(overallTrustFormat, trustFactorName, firstColumnLength+secondColumnLength+1)
	underline := generateUnderline(firstColumnLength + secondColumnLength + 8)

	if trustFactor.trustPercent < 0.5 {
		disgo.Infof(format, underline, trustFactorName, style.Failure(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else if trustFactor.trustPercent < 0.75 {
		disgo.Infof(format, underline, trustFactorName, style.Important(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
	} else {
		disgo.Infof(format, underline, trustFactorName, style.Success(fmt.Sprintf("%4.f%%", trustFactor.trustPercent*100)))
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

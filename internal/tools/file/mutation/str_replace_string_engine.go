package mutation

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type stringReplacementFailureReason int

const (
	stringReplacementFailureNone stringReplacementFailureReason = iota
	stringReplacementFailureMultipleMatches
	stringReplacementFailureNotFound
)

type stringReplacementFailure struct {
	reason     stringReplacementFailureReason
	exactCount int
}

func (f stringReplacementFailure) hasFailure() bool {
	return f.reason != stringReplacementFailureNone
}

type stringReplacementPlan struct {
	newContent          string
	matchStartLine      int
	matchEndLine        int
	replacedEndLine     int
	startLineForDisplay int
	usedNormalizedMatch bool
	actualMatchedLines  []string
	oldLineCount        int
	newLineCount        int
}

type stringReplacementExecution struct {
	plan                stringReplacementPlan
	failure             stringReplacementFailure
	attemptedNormalized bool
}

func buildStringReplacementExecution(oldContent, oldStr, newStr string) stringReplacementExecution {
	exactCount := strings.Count(oldContent, oldStr)
	switch {
	case exactCount == 1:
		return stringReplacementExecution{
			plan: buildExactStringReplacementPlan(oldContent, oldStr, newStr),
		}
	case exactCount > 1:
		return stringReplacementExecution{
			failure: stringReplacementFailure{
				reason:     stringReplacementFailureMultipleMatches,
				exactCount: exactCount,
			},
		}
	default:
		return buildNormalizedStringReplacementExecution(oldContent, oldStr, newStr)
	}
}

func buildExactStringReplacementPlan(oldContent, oldStr, newStr string) stringReplacementPlan {
	matchIdx := strings.Index(oldContent, oldStr)
	matchStartLine := 1 + strings.Count(oldContent[:matchIdx], "\n")
	return buildStringReplacementPlan(
		oldContent,
		oldStr,
		newStr,
		strings.Replace(oldContent, oldStr, newStr, 1),
		matchStartLine,
		matchStartLine+strings.Count(oldStr, "\n"),
		false,
		nil,
	)
}

func buildNormalizedStringReplacementExecution(oldContent, oldStr, newStr string) stringReplacementExecution {
	found, startIdx, endIdx := common.FindWithNormalizedWhitespace(oldContent, oldStr)
	if !found {
		return stringReplacementExecution{
			failure: stringReplacementFailure{
				reason: stringReplacementFailureNotFound,
			},
			attemptedNormalized: true,
		}
	}

	matchStartLine := 1 + strings.Count(oldContent[:startIdx], "\n")
	matchEndLine := 1 + strings.Count(oldContent[:endIdx], "\n")
	actualMatch := oldContent[startIdx : endIdx+1]

	return stringReplacementExecution{
		plan: buildStringReplacementPlan(
			oldContent,
			oldStr,
			newStr,
			oldContent[:startIdx]+newStr+oldContent[endIdx+1:],
			matchStartLine,
			matchEndLine,
			true,
			strings.Split(actualMatch, "\n"),
		),
		attemptedNormalized: true,
	}
}

func buildStringReplacementPlan(oldContent, oldStr, newStr, newContent string, matchStartLine, matchEndLine int, usedNormalized bool, actualMatchedLines []string) stringReplacementPlan {
	return stringReplacementPlan{
		newContent:          newContent,
		matchStartLine:      matchStartLine,
		matchEndLine:        matchEndLine,
		replacedEndLine:     matchStartLine + strings.Count(newStr, "\n"),
		startLineForDisplay: resolveStringReplacementDisplayStartLine(oldContent, oldStr),
		usedNormalizedMatch: usedNormalized,
		actualMatchedLines:  actualMatchedLines,
		oldLineCount:        len(strings.Split(oldStr, "\n")),
		newLineCount:        len(strings.Split(newStr, "\n")),
	}
}

func resolveStringReplacementDisplayStartLine(oldContent, oldStr string) int {
	if idx := strings.Index(oldContent, oldStr); idx >= 0 {
		return strings.Count(oldContent[:idx], "\n") + 1
	}
	return 1
}

func hasNearbyStringReplacementDuplicate(oldContent, newStr string, matchStartLine, matchEndLine int) bool {
	if newStr == "" || matchStartLine <= 0 {
		return false
	}

	nearbyStart := matchStartLine - 10
	if nearbyStart < 1 {
		nearbyStart = 1
	}
	nearbyEnd := matchEndLine + 10
	allLines := strings.Split(oldContent, "\n")
	if nearbyEnd > len(allLines) {
		nearbyEnd = len(allLines)
	}

	beforeContent := ""
	if nearbyStart < matchStartLine {
		beforeContent = strings.Join(allLines[nearbyStart-1:matchStartLine-1], "\n")
	}
	afterContent := ""
	if matchEndLine < nearbyEnd {
		afterContent = strings.Join(allLines[matchEndLine:nearbyEnd], "\n")
	}
	return strings.Contains(beforeContent, newStr) || strings.Contains(afterContent, newStr)
}

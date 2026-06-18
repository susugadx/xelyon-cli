package mutation

import "strings"

type lineRangeReplacementFailureReason int

const (
	lineRangeReplacementFailureNone lineRangeReplacementFailureReason = iota
	lineRangeReplacementFailureMissingRange
	lineRangeReplacementFailureIncompleteRange
	lineRangeReplacementFailureInvalidRange
	lineRangeReplacementFailureEmptyFile
	lineRangeReplacementFailureStartOutOfRange
	lineRangeReplacementFailureEndOutOfRange
)

type lineRangeReplacementFailure struct {
	reason    lineRangeReplacementFailureReason
	startLine int
	endLine   int
	fileLines int
	parseErr  error
}

func (f lineRangeReplacementFailure) hasFailure() bool {
	return f.reason != lineRangeReplacementFailureNone
}

type lineRangeReplacementPlan struct {
	startLine    int
	endLine      int
	oldLineCount int
	newLineCount int
	newContent   string
	beforeRange  string
	lines        []string
}

func (p lineRangeReplacementPlan) replacedEndLine() int {
	return p.startLine + p.newLineCount - 1
}

type lineRangeReplacementExecution struct {
	plan    lineRangeReplacementPlan
	failure lineRangeReplacementFailure
}

func buildLineRangeReplacementExecution(oldContent, newStr, startLineStr, endLineStr string) lineRangeReplacementExecution {
	hasStart := strings.TrimSpace(startLineStr) != ""
	hasEnd := strings.TrimSpace(endLineStr) != ""
	if !hasStart && !hasEnd {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{reason: lineRangeReplacementFailureMissingRange},
		}
	}
	if hasStart != hasEnd {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{reason: lineRangeReplacementFailureIncompleteRange},
		}
	}

	startLine, endLine, err := parseLineRange(startLineStr, endLineStr)
	if err != nil {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{
				reason:   lineRangeReplacementFailureInvalidRange,
				parseErr: err,
			},
		}
	}

	lines := strings.Split(oldContent, "\n")
	if len(lines) == 0 {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{reason: lineRangeReplacementFailureEmptyFile},
		}
	}
	if startLine > len(lines) {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{
				reason:    lineRangeReplacementFailureStartOutOfRange,
				startLine: startLine,
				fileLines: len(lines),
			},
		}
	}
	if endLine > len(lines) {
		return lineRangeReplacementExecution{
			failure: lineRangeReplacementFailure{
				reason:    lineRangeReplacementFailureEndOutOfRange,
				endLine:   endLine,
				fileLines: len(lines),
			},
		}
	}

	newStrLines := strings.Split(newStr, "\n")
	newLines := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newStrLines))
	newLines = append(newLines, lines[:startLine-1]...)
	newLines = append(newLines, newStrLines...)
	newLines = append(newLines, lines[endLine:]...)

	return lineRangeReplacementExecution{
		plan: lineRangeReplacementPlan{
			startLine:    startLine,
			endLine:      endLine,
			oldLineCount: endLine - startLine + 1,
			newLineCount: len(newStrLines),
			newContent:   strings.Join(newLines, "\n"),
			beforeRange:  strings.Join(lines[startLine-1:endLine], "\n"),
			lines:        lines,
		},
	}
}

func hasNearbyLineRangeReplacementDuplicate(lines []string, newStr string, startLine, endLine int) bool {
	if newStr == "" {
		return false
	}

	nearbyStart := startLine - 10
	if nearbyStart < 1 {
		nearbyStart = 1
	}
	nearbyEnd := endLine + 10
	if nearbyEnd > len(lines) {
		nearbyEnd = len(lines)
	}

	beforeContent := ""
	if nearbyStart < startLine {
		beforeContent = strings.Join(lines[nearbyStart-1:startLine-1], "\n")
	}
	afterContent := ""
	if endLine < nearbyEnd {
		afterContent = strings.Join(lines[endLine:nearbyEnd], "\n")
	}
	return strings.Contains(beforeContent, newStr) || strings.Contains(afterContent, newStr)
}

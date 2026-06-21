package replaceengine

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type stringFailureReason int

const (
	stringFailureNone stringFailureReason = iota
	stringFailureMultipleMatches
	stringFailureNotFound
)

// StringFailure は文字列置換 planning の失敗データを表す。
type StringFailure struct {
	reason     stringFailureReason
	exactCount int
}

// HasFailure は失敗データがある場合に true を返す。
func (f StringFailure) HasFailure() bool {
	return f.reason != stringFailureNone
}

// IsMultipleMatches は old_str が複数 exact match した失敗かを返す。
func (f StringFailure) IsMultipleMatches() bool {
	return f.reason == stringFailureMultipleMatches
}

// IsNotFound は exact / normalized のどちらでも old_str が見つからなかった失敗かを返す。
func (f StringFailure) IsNotFound() bool {
	return f.reason == stringFailureNotFound
}

// ExactCount は exact match 数を返す。
func (f StringFailure) ExactCount() int {
	return f.exactCount
}

// StringPlan は文字列置換の pure planning 結果を表す。
type StringPlan struct {
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

// NewContent は置換後の全文を返す。
func (p StringPlan) NewContent() string {
	return p.newContent
}

// MatchStartLine は match 開始行を返す。
func (p StringPlan) MatchStartLine() int {
	return p.matchStartLine
}

// MatchEndLine は match 終了行を返す。
func (p StringPlan) MatchEndLine() int {
	return p.matchEndLine
}

// ReplacedEndLine は置換後 range の終了行を返す。
func (p StringPlan) ReplacedEndLine() int {
	return p.replacedEndLine
}

// StartLineForDisplay は preview 表示の開始行を返す。
func (p StringPlan) StartLineForDisplay() int {
	return p.startLineForDisplay
}

// UsedNormalizedMatch は normalized whitespace fallback が使われた場合に true を返す。
func (p StringPlan) UsedNormalizedMatch() bool {
	return p.usedNormalizedMatch
}

// ActualMatchedLines は normalized match 時に実際に一致した行を返す。
func (p StringPlan) ActualMatchedLines() []string {
	return append([]string(nil), p.actualMatchedLines...)
}

// OldLineCount は置換前 old_str の行数を返す。
func (p StringPlan) OldLineCount() int {
	return p.oldLineCount
}

// NewLineCount は置換後 new_str の行数を返す。
func (p StringPlan) NewLineCount() int {
	return p.newLineCount
}

// StringExecution は文字列置換 planning の成功 plan または失敗データを持つ。
type StringExecution struct {
	plan                StringPlan
	failure             StringFailure
	attemptedNormalized bool
}

// Plan は置換 plan を返す。Failure().HasFailure() が true の場合はゼロ値になる。
func (e StringExecution) Plan() StringPlan {
	return e.plan
}

// Failure は planning の失敗データを返す。
func (e StringExecution) Failure() StringFailure {
	return e.failure
}

// AttemptedNormalized は exact match 失敗後に normalized whitespace match を試した場合に true を返す。
func (e StringExecution) AttemptedNormalized() bool {
	return e.attemptedNormalized
}

// BuildStringExecution は exact / normalized 文字列置換の pure plan を作る。
func BuildStringExecution(oldContent, oldStr, newStr string) StringExecution {
	exactCount := strings.Count(oldContent, oldStr)
	switch {
	case exactCount == 1:
		return StringExecution{
			plan: buildExactStringReplacementPlan(oldContent, oldStr, newStr),
		}
	case exactCount > 1:
		return StringExecution{
			failure: StringFailure{
				reason:     stringFailureMultipleMatches,
				exactCount: exactCount,
			},
		}
	default:
		return buildNormalizedStringReplacementExecution(oldContent, oldStr, newStr)
	}
}

func buildExactStringReplacementPlan(oldContent, oldStr, newStr string) StringPlan {
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

func buildNormalizedStringReplacementExecution(oldContent, oldStr, newStr string) StringExecution {
	found, startIdx, endIdx := common.FindWithNormalizedWhitespace(oldContent, oldStr)
	if !found {
		return StringExecution{
			failure: StringFailure{
				reason: stringFailureNotFound,
			},
			attemptedNormalized: true,
		}
	}

	matchStartLine := 1 + strings.Count(oldContent[:startIdx], "\n")
	matchEndLine := 1 + strings.Count(oldContent[:endIdx], "\n")
	actualMatch := oldContent[startIdx : endIdx+1]

	return StringExecution{
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

func buildStringReplacementPlan(oldContent, oldStr, newStr, newContent string, matchStartLine, matchEndLine int, usedNormalized bool, actualMatchedLines []string) StringPlan {
	return StringPlan{
		newContent:          newContent,
		matchStartLine:      matchStartLine,
		matchEndLine:        matchEndLine,
		replacedEndLine:     matchStartLine + strings.Count(newStr, "\n"),
		startLineForDisplay: resolveStringDisplayStartLine(oldContent, oldStr),
		usedNormalizedMatch: usedNormalized,
		actualMatchedLines:  actualMatchedLines,
		oldLineCount:        len(strings.Split(oldStr, "\n")),
		newLineCount:        len(strings.Split(newStr, "\n")),
	}
}

func resolveStringDisplayStartLine(oldContent, oldStr string) int {
	if idx := strings.Index(oldContent, oldStr); idx >= 0 {
		return strings.Count(oldContent[:idx], "\n") + 1
	}
	return 1
}

// HasNearbyStringDuplicate は対象 range 付近に newStr が既にあるかを判定する。
func HasNearbyStringDuplicate(oldContent, newStr string, matchStartLine, matchEndLine int) bool {
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

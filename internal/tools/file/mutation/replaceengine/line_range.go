package replaceengine

import "strings"

type lineRangeFailureReason int

const (
	lineRangeFailureNone lineRangeFailureReason = iota
	lineRangeFailureMissingRange
	lineRangeFailureIncompleteRange
	lineRangeFailureInvalidRange
	lineRangeFailureEmptyFile
	lineRangeFailureStartOutOfRange
	lineRangeFailureEndOutOfRange
)

// LineRangeFailure は line-range replacement planning の失敗データを表す。
type LineRangeFailure struct {
	reason    lineRangeFailureReason
	startLine int
	endLine   int
	fileLines int
	parseErr  error
}

// HasFailure は失敗データがある場合に true を返す。
func (f LineRangeFailure) HasFailure() bool {
	return f.reason != lineRangeFailureNone
}

// IsMissingRange は start_line/end_line がどちらもない失敗かを返す。
func (f LineRangeFailure) IsMissingRange() bool {
	return f.reason == lineRangeFailureMissingRange
}

// IsIncompleteRange は start_line/end_line の片方だけが指定された失敗かを返す。
func (f LineRangeFailure) IsIncompleteRange() bool {
	return f.reason == lineRangeFailureIncompleteRange
}

// IsInvalidRange は line range parse/validation 失敗かを返す。
func (f LineRangeFailure) IsInvalidRange() bool {
	return f.reason == lineRangeFailureInvalidRange
}

// IsEmptyFile は空ファイルに対する line range 指定かを返す。
func (f LineRangeFailure) IsEmptyFile() bool {
	return f.reason == lineRangeFailureEmptyFile
}

// IsStartOutOfRange は start_line がファイル行数を超えた失敗かを返す。
func (f LineRangeFailure) IsStartOutOfRange() bool {
	return f.reason == lineRangeFailureStartOutOfRange
}

// IsEndOutOfRange は end_line がファイル行数を超えた失敗かを返す。
func (f LineRangeFailure) IsEndOutOfRange() bool {
	return f.reason == lineRangeFailureEndOutOfRange
}

// StartLine は失敗時の start_line を返す。
func (f LineRangeFailure) StartLine() int {
	return f.startLine
}

// EndLine は失敗時の end_line を返す。
func (f LineRangeFailure) EndLine() int {
	return f.endLine
}

// FileLines は対象ファイルの行数を返す。
func (f LineRangeFailure) FileLines() int {
	return f.fileLines
}

// ParseErr は line range parse/validation の error を返す。
func (f LineRangeFailure) ParseErr() error {
	return f.parseErr
}

// LineRangePlan は line-range replacement の pure planning 結果を表す。
type LineRangePlan struct {
	startLine    int
	endLine      int
	oldLineCount int
	newLineCount int
	newContent   string
	beforeRange  string
	lines        []string
}

// StartLine は replacement range の開始行を返す。
func (p LineRangePlan) StartLine() int {
	return p.startLine
}

// EndLine は replacement range の終了行を返す。
func (p LineRangePlan) EndLine() int {
	return p.endLine
}

// OldLineCount は replacement 前の対象行数を返す。
func (p LineRangePlan) OldLineCount() int {
	return p.oldLineCount
}

// NewLineCount は replacement 後の行数を返す。
func (p LineRangePlan) NewLineCount() int {
	return p.newLineCount
}

// NewContent は replacement 後の全文を返す。
func (p LineRangePlan) NewContent() string {
	return p.newContent
}

// BeforeRange は replacement 前の対象 range 文字列を返す。
func (p LineRangePlan) BeforeRange() string {
	return p.beforeRange
}

// Lines は replacement 前の全文行を返す。
func (p LineRangePlan) Lines() []string {
	return append([]string(nil), p.lines...)
}

// ReplacedEndLine は replacement 後 range の終了行を返す。
func (p LineRangePlan) ReplacedEndLine() int {
	return p.startLine + p.newLineCount - 1
}

// LineRangeExecution は line-range replacement の成功 plan または失敗データを持つ。
type LineRangeExecution struct {
	plan    LineRangePlan
	failure LineRangeFailure
}

// Plan は line-range replacement plan を返す。Failure().HasFailure() が true の場合はゼロ値になる。
func (e LineRangeExecution) Plan() LineRangePlan {
	return e.plan
}

// Failure は planning の失敗データを返す。
func (e LineRangeExecution) Failure() LineRangeFailure {
	return e.failure
}

// BuildLineRangeExecution は start_line/end_line replacement の pure plan を作る。
func BuildLineRangeExecution(oldContent, newStr, startLineStr, endLineStr string) LineRangeExecution {
	hasStart := strings.TrimSpace(startLineStr) != ""
	hasEnd := strings.TrimSpace(endLineStr) != ""
	if !hasStart && !hasEnd {
		return LineRangeExecution{
			failure: LineRangeFailure{reason: lineRangeFailureMissingRange},
		}
	}
	if hasStart != hasEnd {
		return LineRangeExecution{
			failure: LineRangeFailure{reason: lineRangeFailureIncompleteRange},
		}
	}

	startLine, endLine, err := ParseLineRange(startLineStr, endLineStr)
	if err != nil {
		return LineRangeExecution{
			failure: LineRangeFailure{
				reason:   lineRangeFailureInvalidRange,
				parseErr: err,
			},
		}
	}

	lines := strings.Split(oldContent, "\n")
	if len(lines) == 0 {
		return LineRangeExecution{
			failure: LineRangeFailure{reason: lineRangeFailureEmptyFile},
		}
	}
	if startLine > len(lines) {
		return LineRangeExecution{
			failure: LineRangeFailure{
				reason:    lineRangeFailureStartOutOfRange,
				startLine: startLine,
				fileLines: len(lines),
			},
		}
	}
	if endLine > len(lines) {
		return LineRangeExecution{
			failure: LineRangeFailure{
				reason:    lineRangeFailureEndOutOfRange,
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

	return LineRangeExecution{
		plan: LineRangePlan{
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

// HasNearbyLineRangeDuplicate は対象 range 付近に newStr が既にあるかを判定する。
func HasNearbyLineRangeDuplicate(lines []string, newStr string, startLine, endLine int) bool {
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

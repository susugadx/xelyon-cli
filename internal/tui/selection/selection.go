package selection

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

// Range は表示列ベースの selection 範囲を表す。
type Range struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// Active は anchor と end が有効な選択範囲を作れるかを返す。
func Active(anchorLine, anchorCol, endLine, endCol int) bool {
	if anchorLine < 0 {
		return false
	}
	return anchorLine != endLine || anchorCol != endCol
}

// Normalize は anchor と end を start <= end の Range に正規化する。
func Normalize(anchorLine, anchorCol, endLine, endCol int) (Range, bool) {
	if anchorLine < 0 {
		return Range{}, false
	}
	r := Range{StartLine: anchorLine, StartCol: anchorCol, EndLine: endLine, EndCol: endCol}
	if r.StartLine > r.EndLine || (r.StartLine == r.EndLine && r.StartCol > r.EndCol) {
		r.StartLine, r.EndLine = r.EndLine, r.StartLine
		r.StartCol, r.EndCol = r.EndCol, r.StartCol
	}
	return r, true
}

// LineRange は2つの line index を start <= end に正規化する。
func LineRange(a, b int) (start, end int) {
	if a <= b {
		return a, b
	}
	return b, a
}

// ColumnsForLine は指定行に selection がかかる表示列範囲を返す。
func ColumnsForLine(r Range, line int, lineCount int) (startCol, endCol int, ok bool) {
	if line < r.StartLine || line > r.EndLine || line >= lineCount {
		return 0, 0, false
	}
	switch {
	case r.StartLine == r.EndLine:
		return r.StartCol, r.EndCol + 1, true
	case line == r.StartLine:
		return r.StartCol, 9999, true
	case line == r.EndLine:
		return 0, r.EndCol + 1, true
	default:
		return 0, 9999, true
	}
}

// ANSIPlainText は ANSI を含む raw lines から selection 範囲の plain text を抽出する。
func ANSIPlainText(lines []string, r Range) (string, int) {
	if len(lines) == 0 || r.EndLine < 0 || r.StartLine >= len(lines) {
		return "", 0
	}
	startLine := maxInt(0, r.StartLine)
	endLine := minInt(len(lines)-1, r.EndLine)
	if startLine > endLine {
		return "", 0
	}

	var result strings.Builder
	lineCount := 0
	for i := startLine; i <= endLine; i++ {
		line := termtext.StripANSI(lines[i])
		runes := []rune(line)
		from := 0
		to := len(runes)

		if i == r.StartLine {
			from = termtext.DisplayColToRuneIndex(line, r.StartCol)
		}
		if i == r.EndLine {
			to = termtext.DisplayColToRuneIndexAfter(line, r.EndCol)
		}
		from = clampInt(from, 0, len(runes))
		to = clampInt(to, 0, len(runes))
		if from > to {
			from = to
		}

		if lineCount > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(string(runes[from:to]))
		lineCount++
	}
	return result.String(), lineCount
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

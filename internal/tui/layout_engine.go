package tui

import (
	"strings"
)

// VisualRow represents a single displayable line on the screen.
type VisualRow struct {
	RawLineIdx  int
	SubRowIdx   int
	Content     string
	Width       int
	PrefixWidth int
}

type Layout struct {
	Rows         []VisualRow
	LineToRowMap []int
	RowToLineMap []int
	Width        int
}

func BuildLayout(rawLines []string, width int) *Layout {
	if width <= 0 {
		width = 80 // fallback
	}
	layout := &Layout{
		Width:        width,
		LineToRowMap: make([]int, len(rawLines)),
	}

	var rows []VisualRow
	var rowToLine []int

	for i, line := range rawLines {
		layout.LineToRowMap[i] = len(rows)
		if line == "" {
			rows = append(rows, VisualRow{RawLineIdx: i, SubRowIdx: 0, Content: "", Width: 0, PrefixWidth: 0})
			rowToLine = append(rowToLine, i)
			continue
		}

		wrapped := wrapLineToVisualRows(line, i, width)
		for _, r := range wrapped {
			rows = append(rows, r)
			rowToLine = append(rowToLine, i)
		}
	}
	layout.Rows = rows
	layout.RowToLineMap = rowToLine
	return layout
}

func wrapLineToVisualRows(line string, rawIdx int, maxWidth int) []VisualRow {
	var rows []VisualRow
	var currentContent strings.Builder
	currentWidth := 0
	subRowIdx := 0
	inEscape := false
	var activeANSI strings.Builder
	var currentANSI strings.Builder

	prefix := getContinuationPrefix(line)
	prefixWidth := 0
	for _, r := range prefix {
		prefixWidth += runeWidth(r)
	}
	if prefixWidth >= maxWidth/2 {
		prefix = ""
		prefixWidth = 0
	}

	for _, r := range line {
		if r == '\033' {
			inEscape = true
			currentContent.WriteRune(r)
			activeANSI.WriteRune(r)
			continue
		}
		if inEscape {
			currentContent.WriteRune(r)
			activeANSI.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
				if r == 'm' {
					code := activeANSI.String()
					if code == "\033[0m" {
						currentANSI.Reset()
					} else {
						currentANSI.WriteString(code)
					}
				}
				activeANSI.Reset()
			}
			continue
		}

		w := runeWidth(r)
		if currentWidth+w > maxWidth && currentWidth > 0 {
			if currentANSI.Len() > 0 {
				currentContent.WriteString("\033[0m")
			}
			pw := 0
			if subRowIdx > 0 {
				pw = prefixWidth
			}
			rows = append(rows, VisualRow{
				RawLineIdx:  rawIdx,
				SubRowIdx:   subRowIdx,
				Content:     currentContent.String(),
				Width:       currentWidth,
				PrefixWidth: pw,
			})
			subRowIdx++
			currentContent.Reset()
			currentWidth = 0
			if currentANSI.Len() > 0 {
				currentContent.WriteString(currentANSI.String())
			}
			if subRowIdx > 0 && prefix != "" {
				currentContent.WriteString(prefix)
				currentWidth += prefixWidth
			}
		}

		currentContent.WriteRune(r)
		currentWidth += w
	}

	if currentContent.Len() > 0 || subRowIdx == 0 {
		if currentANSI.Len() > 0 && !strings.HasSuffix(currentContent.String(), "\033[0m") {
			currentContent.WriteString("\033[0m")
		}
		pw := 0
		if subRowIdx > 0 {
			pw = prefixWidth
		}
		rows = append(rows, VisualRow{
			RawLineIdx:  rawIdx,
			SubRowIdx:   subRowIdx,
			Content:     currentContent.String(),
			Width:       currentWidth,
			PrefixWidth: pw,
		})
	}

	return rows
}

func getContinuationPrefix(line string) string {
	var prefix strings.Builder
	for _, r := range line {
		if r == ' ' {
			prefix.WriteRune(r)
		} else {
			break
		}
	}
	return prefix.String()
}

func (l *Layout) GetVisualCursor(rawLine, rawCol int) (rowIdx, col int) {
	if rawLine < 0 || rawLine >= len(l.LineToRowMap) {
		return -1, -1
	}
	startRow := l.LineToRowMap[rawLine]
	endRow := len(l.Rows)
	if rawLine+1 < len(l.LineToRowMap) {
		endRow = l.LineToRowMap[rawLine+1]
	}
	if startRow >= endRow {
		return -1, -1
	}

	remainingCol := rawCol
	for i := startRow; i < endRow; i++ {
		rowWidth := l.Rows[i].Width
		prefixWidth := l.Rows[i].PrefixWidth

		contentWidth := rowWidth - prefixWidth
		if remainingCol <= contentWidth {
			if remainingCol < contentWidth || i == endRow-1 {
				return i, prefixWidth + remainingCol
			}
		}
		remainingCol -= contentWidth
	}
	return endRow - 1, l.Rows[endRow-1].Width
}

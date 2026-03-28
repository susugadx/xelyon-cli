package tui

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Layout engine processes raw lines to visual rows.

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

	flushRow := func() {
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

	gr := uniseg.NewGraphemes(line)
	for gr.Next() {
		cluster := gr.Str()
		firstRune := gr.Runes()[0]

		// ANSIエスケープシーケンスの処理
		// エスケープ文字はASCIIなので常に独立したgraphemeクラスタになる
		if firstRune == '\033' {
			inEscape = true
			currentContent.WriteString(cluster)
			activeANSI.WriteString(cluster)
			continue
		}
		if inEscape {
			currentContent.WriteString(cluster)
			activeANSI.WriteString(cluster)
			if (firstRune >= 'A' && firstRune <= 'Z') || (firstRune >= 'a' && firstRune <= 'z') {
				inEscape = false
				if firstRune == 'm' {
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

		if cluster == "\t" {
			tabRemaining := visualTabWidth
			for tabRemaining > 0 {
				if currentWidth >= maxWidth && currentWidth > 0 {
					flushRow()
				}
				room := maxWidth - currentWidth
				if room <= 0 {
					room = maxWidth
				}
				chunk := min(tabRemaining, room)
				currentContent.WriteString(strings.Repeat(" ", chunk))
				currentWidth += chunk
				tabRemaining -= chunk

				if tabRemaining > 0 && currentWidth >= maxWidth {
					flushRow()
				}
			}
			continue
		}

		w := plainTextDisplayWidth(cluster)
		if currentWidth+w > maxWidth && currentWidth > 0 {
			flushRow()
		}

		currentContent.WriteString(cluster)
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

func (l *Layout) GetRawColumnForVisualRow(visRowIdx int) int {
	if visRowIdx < 0 || visRowIdx >= len(l.Rows) {
		return 0
	}
	rawLine := l.Rows[visRowIdx].RawLineIdx
	startRow := l.LineToRowMap[rawLine]
	rawCol := 0
	for i := startRow; i < visRowIdx; i++ {
		rawCol += l.Rows[i].Width - l.Rows[i].PrefixWidth
	}
	return rawCol
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

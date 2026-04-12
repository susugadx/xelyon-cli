package tui

import (
	"strings"

	"github.com/rivo/uniseg"
)

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
		rows = append(rows, VisualRow{
			RawLineIdx:  rawIdx,
			SubRowIdx:   subRowIdx,
			Content:     currentContent.String(),
			Width:       currentWidth,
			PrefixWidth: continuationPrefixWidth(subRowIdx, prefixWidth),
		})
		subRowIdx++
		currentContent.Reset()
		currentWidth = 0
		if currentANSI.Len() > 0 {
			currentContent.WriteString(currentANSI.String())
		}
		if prefix != "" {
			appendContinuationPrefix(&currentContent, &currentWidth, subRowIdx, prefix, prefixWidth)
		}
	}

	gr := uniseg.NewGraphemes(line)
	for gr.Next() {
		cluster := gr.Str()
		firstRune := gr.Runes()[0]

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
			currentWidth = appendWrappedTab(&rows, &currentContent, currentWidth, &subRowIdx, rawIdx, maxWidth, prefix, prefixWidth, currentANSI.String())
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
		rows = append(rows, VisualRow{
			RawLineIdx:  rawIdx,
			SubRowIdx:   subRowIdx,
			Content:     currentContent.String(),
			Width:       currentWidth,
			PrefixWidth: continuationPrefixWidth(subRowIdx, prefixWidth),
		})
	}

	return rows
}

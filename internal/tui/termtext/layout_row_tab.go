package termtext

import "strings"

func appendWrappedTab(rows *[]VisualRow, currentContent *strings.Builder, currentWidth int, subRowIdx *int, rawIdx int, maxWidth int, prefix string, prefixWidth int, currentANSI string) int {
	tabRemaining := VisualTabWidth
	flushRow := func() {
		if currentANSI != "" {
			currentContent.WriteString("\033[0m")
		}
		*rows = append(*rows, VisualRow{
			RawLineIdx:  rawIdx,
			SubRowIdx:   *subRowIdx,
			Content:     currentContent.String(),
			Width:       currentWidth,
			PrefixWidth: continuationPrefixWidth(*subRowIdx, prefixWidth),
		})
		*subRowIdx = *subRowIdx + 1
		currentContent.Reset()
		currentWidth = 0
		if currentANSI != "" {
			currentContent.WriteString(currentANSI)
		}
		if prefix != "" {
			appendContinuationPrefix(currentContent, &currentWidth, *subRowIdx, prefix, prefixWidth)
		}
	}

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
	return currentWidth
}

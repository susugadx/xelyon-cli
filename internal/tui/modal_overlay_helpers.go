package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func promptOverlayBaseLines(base string, width int, height int) []string {
	lines := strings.Split(base, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, line := range lines {
		lines[i] = termtext.FillANSITextWidth(line, width, "")
	}
	return lines
}

func promptOverlayPanelWidth(screenWidth int) int {
	if screenWidth <= 0 {
		return 0
	}
	if screenWidth <= 24 {
		return screenWidth
	}
	return min(88, screenWidth-4)
}

func renderPromptOverlayLine(baseLine string, panelLine string, left int, panelWidth int, width int) string {
	plain := termtext.StripANSI(baseLine)
	leftPart := termtext.FillANSITextWidth(promptDisplaySegment(plain, 0, left), left, "")
	panel := termtext.FillANSITextWidth(panelLine, panelWidth, theme.Config.BgNormal)
	rightWidth := max(0, width-left-panelWidth)
	rightPart := termtext.FillANSITextWidth(promptDisplaySegment(plain, left+panelWidth, rightWidth), rightWidth, "")
	return leftPart + panel + rightPart
}

func promptDisplaySegment(text string, start int, width int) string {
	if width <= 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	runes := []rune(text)
	startIdx := termtext.DisplayColToRuneIndex(text, start)
	endIdx := termtext.DisplayColToRuneIndex(text, start+width)
	if startIdx > len(runes) {
		return ""
	}
	if endIdx > len(runes) {
		endIdx = len(runes)
	}
	if endIdx < startIdx {
		return ""
	}
	return string(runes[startIdx:endIdx])
}

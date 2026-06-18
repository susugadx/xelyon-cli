package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

func (m Model) renderPromptOverlay(base string) string {
	if m.prompt == nil {
		return base
	}

	width := max(1, m.width)
	height := max(1, m.height)
	baseLines := promptOverlayBaseLines(base, width, height)
	panelWidth := promptOverlayPanelWidth(width)
	if panelWidth <= 0 {
		return strings.Join(baseLines, "\n")
	}

	panelLines := m.promptPanelLines(panelWidth)
	top := max(0, (height-len(panelLines))/2)
	left := max(0, (width-panelWidth)/2)
	for i, line := range panelLines {
		row := top + i
		if row >= height {
			break
		}
		baseLines[row] = renderPromptOverlayLine(baseLines[row], line, left, panelWidth, width)
	}
	return strings.Join(baseLines, "\n")
}

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

func (m Model) promptPanelLines(width int) []string {
	palette := theme.Config
	lines := m.promptContentLines(width)
	for i, line := range lines {
		lines[i] = termtext.FillANSITextWidth(line, width, palette.BgNormal)
	}
	return lines
}

func (m Model) promptContentLines(width int) []string {
	palette := theme.Config
	title := termtext.StripANSI(termtext.SanitizeSingleLineANSI(m.prompt.req.Title))
	if title == "" {
		title = promptDefaultTitle(m.prompt.req.Kind)
	}
	message := termtext.StripANSI(termtext.SanitizeSingleLineANSI(m.prompt.req.Message))
	lines := []string{
		palette.BgHeader + palette.Bold + palette.FgBright + "  " + fitPlainPromptText(title, width-4) + palette.Reset,
		palette.BgNormal + palette.FgNormal + "  " + fitPlainPromptText(message, width-4) + palette.Reset,
		"",
	}

	if m.prompt.mode == promptModalText {
		view := m.prompt.text.viewLine()
		lines = append(lines,
			palette.BgSelected+palette.FgBright+"  "+fitPlainPromptText(view, width-4)+palette.Reset,
			palette.BgNormal+palette.FgDim+"  Enter:submit  Esc:cancel"+palette.Reset,
		)
		return lines
	}

	options := promptOptions(m.prompt.req)
	for i, opt := range options {
		selected := i == m.prompt.selected
		prefix := "  ( ) "
		bg := palette.BgNormal
		fg := palette.FgNormal
		if selected {
			prefix = "  (*) "
			bg = palette.BgSelected
			fg = palette.FgBright
		}
		if m.prompt.req.Kind == uiprompt.PromptKindMultiChoice {
			if m.prompt.values[opt.value] {
				prefix = "  [x] "
			} else {
				prefix = "  [ ] "
			}
			if selected {
				bg = palette.BgSelected
				fg = palette.FgBright
			}
		}
		label := opt.label
		if opt.description != "" {
			label += " - " + opt.description
		}
		lines = append(lines, bg+fg+prefix+fitPlainPromptText(label, width-len(prefix)-2)+palette.Reset)
	}

	hint := "  Up/Down/j/k:move  Enter:submit  Esc:cancel"
	if m.prompt.req.Kind == uiprompt.PromptKindMultiChoice {
		hint = "  Up/Down/j/k:move  Space:toggle  Enter:submit  Esc:cancel"
	}
	lines = append(lines, "", palette.BgNormal+palette.FgDim+fitPlainPromptText(hint, width)+palette.Reset)
	return lines
}

func promptDefaultTitle(kind uiprompt.PromptKind) string {
	switch kind {
	case uiprompt.PromptKindConfirm:
		return "Confirm"
	case uiprompt.PromptKindSingleChoice, uiprompt.PromptKindMultiChoice:
		return "Choose"
	case uiprompt.PromptKindText:
		return "Input"
	default:
		return "Prompt"
	}
}

func fitPlainPromptText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if termtext.PlainTextDisplayWidth(text) <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 && termtext.PlainTextDisplayWidth(string(runes)+"...") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

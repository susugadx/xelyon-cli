package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) reviewView() string {
	if m.reviewScreen == nil {
		return "Loading..."
	}

	bodyHeight := m.reviewBodyHeight()
	bodyLines := m.reviewBodyLines()
	header := m.renderReviewHeader(m.width)
	body := m.renderReviewBody(bodyLines, bodyHeight)
	status := m.renderReviewStatus(m.width, reviewBodyScrollBoundsForLines(bodyLines, bodyHeight))

	return header + "\n" + body + "\n" + status
}

func (m Model) reviewBodyHeight() int {
	return max(1, m.height-2)
}

func (m Model) renderReviewHeader(width int) string {
	titleText := "Review"
	title := theme.Config.BgHeader + theme.Config.Bold + theme.Config.FgBright + " " + titleText + " " + theme.Config.Reset
	hintText := reviewHeaderHint(m.reviewScreen)
	hint := theme.Config.FgDim + hintText + theme.Config.Reset
	padding := width - lipgloss.Width(titleText) - 2 - lipgloss.Width(hintText)
	if padding < 2 {
		return termtext.FillANSITextWidth(title, width, theme.Config.BgHeader)
	}
	return termtext.FillANSITextWidth(title+strings.Repeat(" ", padding)+hint, width, theme.Config.BgHeader)
}

func reviewHeaderHint(rs *reviewScreen) string {
	if rs == nil {
		return ""
	}
	switch rs.mode {
	case reviewScreenCustom:
		return "Esc:presets"
	case reviewScreenSubmitted:
		return "Esc:back"
	default:
		return "Esc:back"
	}
}

func (m Model) renderReviewBody(lines []string, height int) string {
	return m.reviewScreen.bodyViewport.render(lines, height, m.width)
}

func (m Model) reviewBodyLines() []string {
	rs := m.reviewScreen
	switch rs.mode {
	case reviewScreenCustom:
		return m.reviewCustomLines(rs)
	case reviewScreenSubmitted:
		return reviewSubmittedLines(rs)
	default:
		return reviewPresetLines(rs)
	}
}

func reviewPresetLines(rs *reviewScreen) []string {
	lines := []string{
		theme.Config.BgNormal + theme.Config.FgDim + " Select review preset" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + "" + theme.Config.Reset,
	}
	for i, preset := range reviewPresets {
		prefix := "  "
		style := theme.Config.BgNormal + theme.Config.FgNormal
		if i == rs.presetIndex {
			prefix = "> "
			style = theme.Config.BgSelected + theme.Config.FgBright
		}
		lines = append(lines, style+" "+prefix+preset.label+theme.Config.Reset)
	}
	return lines
}

func (m Model) reviewCustomLines(rs *reviewScreen) []string {
	inputView := strings.ReplaceAll(rs.customInput.View(), theme.Config.Reset, theme.Config.Reset+theme.Config.BgNormal)
	return []string{
		theme.Config.BgNormal + theme.Config.FgDim + " Custom review instructions" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgDim + "" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgBright + "  " + inputView + theme.Config.Reset,
	}
}

func reviewSubmittedLines(rs *reviewScreen) []string {
	lines := []string{
		reviewStateLine(rs),
	}
	if rs.request == nil {
		return lines
	}
	lines = append(lines,
		theme.Config.BgNormal+theme.Config.FgDim+" Target: "+string(rs.request.TargetKind)+theme.Config.Reset,
	)
	if rs.request.CustomInstructions != "" {
		lines = append(lines,
			theme.Config.BgNormal+theme.Config.FgDim+" Custom instructions: "+termtext.SanitizeSingleLineANSI(rs.request.CustomInstructions)+theme.Config.Reset,
		)
	}
	if rs.runState == reviewRunFailed && rs.errMessage != "" {
		lines = append(lines,
			theme.Config.BgNormal+theme.Config.FgRed+" Error: "+termtext.SanitizeSingleLineANSI(rs.errMessage)+theme.Config.Reset,
		)
	}
	if rs.runState == reviewRunSucceeded && rs.report != nil {
		lines = append(lines, reviewReportLines(*rs.report)...)
	}
	return lines
}

func reviewStateLine(rs *reviewScreen) string {
	style := theme.Config.FgYellow
	switch rs.runState {
	case reviewRunSucceeded:
		style = theme.Config.FgGreen
	case reviewRunFailed:
		style = theme.Config.FgRed
	}
	return theme.Config.BgNormal + style + " " + rs.message + theme.Config.Reset
}

func (m Model) renderReviewStatus(width int, bodyBounds reviewBodyScrollBounds) string {
	rs := m.reviewScreen
	leftText := reviewStatusText(rs)
	left := " " + theme.Config.FgGreen + leftText + theme.Config.Reset
	hintText := reviewStatusHint(rs, bodyBounds)
	right := theme.Config.FgDim + hintText + theme.Config.Reset + " "
	padding := width - lipgloss.Width(leftText) - lipgloss.Width(hintText) - 3
	if padding < 1 {
		return termtext.FillANSITextWidth(left, width, "")
	}
	return fitANSITextWidth(left+strings.Repeat(" ", padding)+right, width)
}

func reviewStatusText(rs *reviewScreen) string {
	if rs == nil {
		return "review"
	}
	if rs.mode == reviewScreenSubmitted {
		switch rs.runState {
		case reviewRunRunning:
			return "review running"
		case reviewRunSucceeded:
			return "review complete"
		case reviewRunFailed:
			return "review failed"
		default:
			return "review pending"
		}
	}
	return "review"
}

func reviewStatusHint(rs *reviewScreen, bodyBounds reviewBodyScrollBounds) string {
	if rs == nil {
		return ""
	}
	switch rs.mode {
	case reviewScreenCustom:
		return "Enter:confirm  Esc:presets"
	case reviewScreenSubmitted:
		if rs.runState == reviewRunRunning {
			return "Esc:cancel"
		}
		if bodyBounds.hasOverflow() {
			return "j/k:scroll  PgUp/PgDn:page  Esc:back"
		}
		return "Esc:back"
	default:
		return "j/k:move  Enter:select  Esc:back"
	}
}

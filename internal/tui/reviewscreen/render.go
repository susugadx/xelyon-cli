package reviewscreen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// View は review screen 全体を描画する。
func (rs *Screen) View(width, height int) string {
	if rs == nil {
		return "Loading..."
	}

	bodyHeight := reviewBodyHeight(height)
	bodyLines := rs.bodyLines()
	header := renderHeader(rs, width)
	body := rs.bodyViewport.render(bodyLines, bodyHeight, width)
	status := renderStatus(rs, width)

	return header + "\n" + body + "\n" + status
}

func reviewBodyHeight(height int) int {
	return max(1, height-2)
}

func renderHeader(rs *Screen, width int) string {
	titleText := "Review"
	title := theme.Config.BgHeader + theme.Config.Bold + theme.Config.FgBright + " " + titleText + " " + theme.Config.Reset
	hintText := reviewHeaderHint(rs)
	hint := theme.Config.FgDim + hintText + theme.Config.Reset
	padding := width - lipgloss.Width(titleText) - 2 - lipgloss.Width(hintText)
	if padding < 2 {
		return termtext.FillANSITextWidth(title, width, theme.Config.BgHeader)
	}
	return termtext.FillANSITextWidth(title+strings.Repeat(" ", padding)+hint, width, theme.Config.BgHeader)
}

func reviewHeaderHint(rs *Screen) string {
	if rs == nil {
		return ""
	}
	switch rs.mode {
	case ModeCustom:
		return "Esc:presets"
	default:
		return "Esc:back"
	}
}

func (rs *Screen) bodyLines() []string {
	switch rs.mode {
	case ModeCustom:
		return reviewCustomLines(rs)
	default:
		return reviewPresetLines(rs)
	}
}

func reviewPresetLines(rs *Screen) []string {
	lines := []string{
		theme.Config.BgNormal + theme.Config.FgDim + " Select review preset" + theme.Config.Reset,
	}
	lines = appendReviewNoticeLine(lines, rs)
	lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+""+theme.Config.Reset)
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

func reviewCustomLines(rs *Screen) []string {
	inputView := strings.ReplaceAll(rs.customInput.View(), theme.Config.Reset, theme.Config.Reset+theme.Config.BgNormal)
	lines := []string{
		theme.Config.BgNormal + theme.Config.FgDim + " Review current changes with custom focus" + theme.Config.Reset,
		theme.Config.BgNormal + theme.Config.FgBright + "  " + inputView + theme.Config.Reset,
	}
	lines = appendReviewNoticeLine(lines, rs)
	lines = append(lines,
		theme.Config.BgNormal+theme.Config.FgDim+""+theme.Config.Reset,
		theme.Config.BgNormal+theme.Config.FgDim+"  Reviews all current changes."+theme.Config.Reset,
		theme.Config.BgNormal+theme.Config.FgDim+"  Custom focus adjusts priorities; it does not narrow files or diff scope."+theme.Config.Reset,
		theme.Config.BgNormal+theme.Config.FgDim+"  It is not a single-finding recheck mode."+theme.Config.Reset,
	)
	return lines
}

func appendReviewNoticeLine(lines []string, rs *Screen) []string {
	if rs == nil || rs.notice == "" {
		return lines
	}
	return append(lines, theme.Config.BgNormal+theme.Config.FgYellow+"  "+rs.notice+theme.Config.Reset)
}

func renderStatus(rs *Screen, width int) string {
	leftText := reviewStatusText(rs)
	left := " " + theme.Config.FgGreen + leftText + theme.Config.Reset
	hintText := reviewStatusHint(rs)
	right := theme.Config.FgDim + hintText + theme.Config.Reset + " "
	padding := width - lipgloss.Width(leftText) - lipgloss.Width(hintText) - 3
	if padding < 1 {
		return termtext.FillANSITextWidth(left, width, "")
	}
	return termtext.FillANSITextWidth(left+strings.Repeat(" ", padding)+right, width, "")
}

func reviewStatusText(rs *Screen) string {
	if rs == nil {
		return "review"
	}
	return "review"
}

func reviewStatusHint(rs *Screen) string {
	if rs == nil {
		return ""
	}
	switch rs.mode {
	case ModeCustom:
		return "Enter:confirm  Esc:presets"
	default:
		return "j/k:move  Enter:select  Esc:back"
	}
}

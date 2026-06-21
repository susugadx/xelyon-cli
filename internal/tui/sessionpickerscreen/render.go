package sessionpickerscreen

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// PanelLines は session picker overlay の panel 行を返す。
func (p *Screen) PanelLines(width int, height int) []string {
	palette := theme.Config
	title := "Resume"
	if p.all {
		title = "Resume - all sessions"
	}
	lines := []string{
		palette.BgHeader + palette.Bold + palette.FgBright + "  " + fitPlainPromptText(title, width-4) + palette.Reset,
	}

	if p.filtering {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  Filter: "+fitPlainPromptText(p.filter, width-12)+palette.Reset)
	} else {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  /:filter  j/k/Up/Down:move  Enter:resume  Esc:cancel"+palette.Reset)
	}
	lines = append(lines, "")
	rows := p.rows()
	if len(rows) == 0 {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  No sessions"+palette.Reset)
		return fillLines(lines, width)
	}

	start, end := rowWindow(len(rows), p.selected, height)
	for i := start; i < end; i++ {
		lines = append(lines, selectableLine(i == p.selected, sessionPickerLabel(rows[i]), width))
	}
	return fillLines(lines, width)
}

func sessionPickerLabel(row Candidate) string {
	timestamp := row.LastModified.Format("2006-01-02 15:04")
	id := row.ID
	if len(id) > 8 {
		id = id[:8]
	}
	model := strings.TrimSpace(row.Model)
	provider := strings.TrimSpace(row.ProviderName)
	runtime := model
	if provider != "" && model != "" {
		runtime = provider + "/" + model
	} else if provider != "" {
		runtime = provider
	}
	preview := strings.TrimSpace(row.Preview)
	if preview == "" {
		preview = "(no preview)"
	}
	cwd := filepath.Base(strings.TrimSpace(row.WorkingDir))
	if cwd == "." || cwd == "/" {
		cwd = strings.TrimSpace(row.WorkingDir)
	}
	parts := []string{timestamp, id}
	if runtime != "" {
		parts = append(parts, runtime)
	}
	if cwd != "" {
		parts = append(parts, cwd)
	}
	parts = append(parts, preview)
	return strings.Join(parts, "  ")
}

func fillLines(lines []string, width int) []string {
	palette := theme.Config
	for i, line := range lines {
		lines[i] = termtext.FillANSITextWidth(line, width, palette.BgNormal)
	}
	return lines
}

func selectableLine(selected bool, text string, width int) string {
	palette := theme.Config
	bg := palette.BgNormal
	fg := palette.FgNormal
	prefix := "  "
	if selected {
		bg = palette.BgSelected
		fg = palette.FgBright
		prefix = "> "
	}
	return bg + fg + prefix + fitPlainPromptText(text, width-len(prefix)-1) + palette.Reset
}

func fitPlainPromptText(text string, width int) string {
	return termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(text), max(0, width))
}

func rowWindow(rowCount int, selected int, height int) (int, int) {
	limit := visibleRows(rowCount, height)
	if limit >= rowCount {
		return 0, rowCount
	}
	start := selected - limit + 1
	if start < 0 {
		start = 0
	}
	if start+limit > rowCount {
		start = rowCount - limit
	}
	return start, start + limit
}

func visibleRows(rowCount int, height int) int {
	maxRows := max(3, height-10)
	if rowCount < maxRows {
		return rowCount
	}
	return maxRows
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

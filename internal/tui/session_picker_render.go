package tui

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderSessionPickerOverlay(base string) string {
	if m.sessionPicker == nil {
		return base
	}

	width := max(1, m.width)
	height := max(1, m.height)
	baseLines := promptOverlayBaseLines(base, width, height)
	panelWidth := promptOverlayPanelWidth(width)
	if panelWidth <= 0 {
		return strings.Join(baseLines, "\n")
	}

	panelLines := m.sessionPickerPanelLines(panelWidth)
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

func (m Model) sessionPickerPanelLines(width int) []string {
	palette := theme.Config
	title := "Resume"
	if m.sessionPicker.all {
		title = "Resume - all sessions"
	}
	lines := []string{
		palette.BgHeader + palette.Bold + palette.FgBright + "  " + fitPlainPromptText(title, width-4) + palette.Reset,
	}

	if m.sessionPicker.filtering {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  Filter: "+fitPlainPromptText(m.sessionPicker.filter, width-12)+palette.Reset)
	} else {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  /:filter  j/k/Up/Down:move  Enter:resume  Esc:cancel"+palette.Reset)
	}
	lines = append(lines, "")
	rows := m.sessionPicker.rows()
	if len(rows) == 0 {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  No sessions"+palette.Reset)
		return providerPickerFillLines(lines, width)
	}

	start, end := providerPickerRowWindow(len(rows), m.sessionPicker.selected, m.height)
	for i := start; i < end; i++ {
		lines = append(lines, providerPickerSelectableLine(i == m.sessionPicker.selected, sessionPickerLabel(rows[i]), width))
	}
	return providerPickerFillLines(lines, width)
}

func sessionPickerLabel(row SessionCandidate) string {
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

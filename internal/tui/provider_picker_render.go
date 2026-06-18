package tui

import "strings"

func (m Model) renderProviderPickerOverlay(base string) string {
	if m.providerPicker == nil {
		return base
	}

	width := max(1, m.width)
	height := max(1, m.height)
	baseLines := promptOverlayBaseLines(base, width, height)
	panelWidth := promptOverlayPanelWidth(width)
	if panelWidth <= 0 {
		return strings.Join(baseLines, "\n")
	}

	panelLines := m.providerPicker.PanelLines(panelWidth, height)
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

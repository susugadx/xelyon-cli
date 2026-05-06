package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

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

	panelLines := m.providerPickerPanelLines(panelWidth)
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

func (m Model) providerPickerPanelLines(width int) []string {
	palette := theme.Config
	title := m.providerPickerTitle()
	lines := []string{
		palette.BgHeader + palette.Bold + palette.FgBright + "  " + fitPlainPromptText(title, width-4) + palette.Reset,
	}

	if m.providerPicker.mode == providerPickerCustom {
		lines = append(lines,
			palette.BgNormal+palette.FgNormal+"  "+fitPlainPromptText("Custom value", width-4)+palette.Reset,
			"",
			palette.BgSelected+palette.FgBright+"  "+fitPlainPromptText(providerPickerInputView(m.providerPicker.customInput), width-4)+palette.Reset,
			palette.BgNormal+palette.FgDim+"  Enter:apply  Esc:back"+palette.Reset,
		)
		return providerPickerFillLines(lines, width)
	}

	if m.providerPicker.filtering {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  Filter: "+fitPlainPromptText(m.providerPicker.filter, width-12)+palette.Reset)
	} else {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  /:filter  j/k/Up/Down:move  Enter:select  Esc:cancel"+palette.Reset)
	}
	lines = append(lines, "")

	switch m.providerPicker.mode {
	case providerPickerProviders:
		rows := m.providerPicker.providerRows()
		lines = append(lines, m.providerPickerProviderLines(rows, width)...)
	case providerPickerModels:
		rows := m.providerPicker.modelRows()
		lines = append(lines, m.providerPickerModelLines(rows, width)...)
		backHint := "  Backspace:providers"
		if m.providerPicker.step == providerPickerStepAzureCatalogModelSelect {
			backHint = "  Backspace:deployments"
		}
		if m.providerPicker.currentOnly {
			backHint = "  Backspace:cancel"
		}
		lines = append(lines, "", palette.BgNormal+palette.FgDim+backHint+palette.Reset)
	}

	return providerPickerFillLines(lines, width)
}

func (m Model) providerPickerTitle() string {
	switch m.providerPicker.mode {
	case providerPickerProviders:
		return "Provider"
	case providerPickerModels:
		if m.providerPicker.step == providerPickerStepAzureCatalogModelSelect {
			return "Catalog model"
		}
		label := m.providerPicker.providerLabel
		if label == "" {
			label = m.providerPicker.provider
		}
		if label == "" {
			label = "Current provider"
		}
		return "Model - " + label
	case providerPickerCustom:
		if m.providerPicker.step == providerPickerStepAzureDeploymentInput {
			return "Custom deployment"
		}
		if m.providerPicker.step == providerPickerStepAzureCatalogModelCustom {
			return "Custom catalog model"
		}
		if m.providerPicker.provider == "azure" {
			return "Custom deployment"
		}
		return "Custom model"
	default:
		return "Provider"
	}
}

func (m Model) providerPickerProviderLines(rows []providerpicker.ProviderCandidate, width int) []string {
	if len(rows) == 0 {
		return []string{theme.Config.BgNormal + theme.Config.FgDim + "  No providers" + theme.Config.Reset}
	}
	start, end := providerPickerRowWindow(len(rows), m.providerPicker.selected, m.height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, providerPickerSelectableLine(i == m.providerPicker.selected, providerPickerProviderLabel(rows[i]), width))
	}
	return lines
}

func providerPickerProviderLabel(row providerpicker.ProviderCandidate) string {
	label := row.Label
	if label == "" {
		label = row.Key
	}
	if row.Key != "" && row.Key != label {
		label += " (" + row.Key + ")"
	}
	label += "  " + string(row.CredentialStatus)
	if row.Current {
		label += "  current"
	}
	return label
}

func (m Model) providerPickerModelLines(rows []providerpicker.ModelCandidate, width int) []string {
	if len(rows) == 0 {
		return []string{theme.Config.BgNormal + theme.Config.FgDim + "  No models" + theme.Config.Reset}
	}
	start, end := providerPickerRowWindow(len(rows), m.providerPicker.selected, m.height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, providerPickerSelectableLine(i == m.providerPicker.selected, providerPickerModelLabel(rows[i]), width))
	}
	return lines
}

func providerPickerModelLabel(row providerpicker.ModelCandidate) string {
	label := row.Name
	tags := make([]string, 0, 3)
	if row.Current {
		tags = append(tags, "current")
	}
	if row.Default {
		tags = append(tags, "default")
	}
	if row.Custom {
		tags = append(tags, "custom")
	}
	if len(tags) > 0 {
		label += "  " + strings.Join(tags, " ")
	}
	return label
}

func providerPickerSelectableLine(selected bool, text string, width int) string {
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

func providerPickerFillLines(lines []string, width int) []string {
	palette := theme.Config
	for i, line := range lines {
		lines[i] = termtext.FillANSITextWidth(line, width, palette.BgNormal)
	}
	return lines
}

func providerPickerInputView(input textinput.Model) string {
	return termtext.StripANSI(termtext.SanitizeSingleLineANSI(input.View()))
}

func providerPickerVisibleRows(rowCount int, height int) int {
	maxRows := max(3, height-10)
	if rowCount < maxRows {
		return rowCount
	}
	return maxRows
}

func providerPickerRowWindow(rowCount int, selected int, height int) (int, int) {
	limit := providerPickerVisibleRows(rowCount, height)
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

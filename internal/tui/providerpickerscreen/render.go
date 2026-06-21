package providerpickerscreen

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// PanelLines は provider picker overlay の panel 行を返す。
func (p *Screen) PanelLines(width int, height int) []string {
	palette := theme.Config
	title := p.title()
	lines := []string{
		palette.BgHeader + palette.Bold + palette.FgBright + "  " + fitPlainPromptText(title, width-4) + palette.Reset,
	}

	if p.mode == ModeCustom {
		lines = append(lines,
			palette.BgNormal+palette.FgNormal+"  "+fitPlainPromptText("Custom value", width-4)+palette.Reset,
			"",
			palette.BgSelected+palette.FgBright+"  "+fitPlainPromptText(providerPickerInputView(p.customInput), width-4)+palette.Reset,
			palette.BgNormal+palette.FgDim+"  Enter:apply  Esc:back"+palette.Reset,
		)
		return providerPickerFillLines(lines, width)
	}

	if p.filtering {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  Filter: "+fitPlainPromptText(p.filter, width-12)+palette.Reset)
	} else {
		lines = append(lines, palette.BgNormal+palette.FgDim+"  /:filter  j/k/Up/Down:move  Enter:select  Esc:cancel"+palette.Reset)
	}
	lines = append(lines, "")

	switch p.mode {
	case ModeProviders:
		rows := p.providerRows()
		lines = append(lines, p.providerLines(rows, width, height)...)
	case ModeModels:
		rows := p.modelRows()
		lines = append(lines, p.modelLines(rows, width, height)...)
		backHint := "  Backspace:providers"
		if p.step == StepAzureCatalogModelSelect {
			backHint = "  Backspace:deployments"
		}
		if p.currentOnly {
			backHint = "  Backspace:cancel"
		}
		lines = append(lines, "", palette.BgNormal+palette.FgDim+backHint+palette.Reset)
	}

	return providerPickerFillLines(lines, width)
}

func (p *Screen) title() string {
	switch p.mode {
	case ModeProviders:
		return "Provider"
	case ModeModels:
		if p.step == StepAzureCatalogModelSelect {
			return "Catalog model"
		}
		label := p.providerLabel
		if label == "" {
			label = p.provider
		}
		if label == "" {
			label = "Current provider"
		}
		return "Model - " + label
	case ModeCustom:
		if p.step == StepAzureDeploymentInput {
			return "Custom deployment"
		}
		if p.step == StepAzureCatalogModelCustom {
			return "Custom catalog model"
		}
		if p.provider == "azure" {
			return "Custom deployment"
		}
		return "Custom model"
	default:
		return "Provider"
	}
}

func (p *Screen) providerLines(rows []providerpicker.ProviderCandidate, width int, height int) []string {
	if len(rows) == 0 {
		return []string{theme.Config.BgNormal + theme.Config.FgDim + "  No providers" + theme.Config.Reset}
	}
	start, end := providerPickerRowWindow(len(rows), p.selected, height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, providerPickerSelectableLine(i == p.selected, providerPickerProviderLabel(rows[i]), width))
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

func (p *Screen) modelLines(rows []providerpicker.ModelCandidate, width int, height int) []string {
	if len(rows) == 0 {
		return []string{theme.Config.BgNormal + theme.Config.FgDim + "  No models" + theme.Config.Reset}
	}
	start, end := providerPickerRowWindow(len(rows), p.selected, height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, providerPickerSelectableLine(i == p.selected, providerPickerModelLabel(rows[i]), width))
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

func fitPlainPromptText(text string, width int) string {
	return termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(text), max(0, width))
}

func providerPickerVisibleRows(rowCount int, height int) int {
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

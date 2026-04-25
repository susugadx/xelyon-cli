package tui

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderConfigDetailPane は右ペイン（詳細/エディタ）を構築する。
func (m Model) renderConfigDetailPane(width, height int) []string {
	if width <= 0 {
		return nil
	}

	field := m.configScreen.selectedField()
	if field == nil {
		return appendConfigPanePadding(
			[]string{termtext.FillANSITextWidth(theme.Config.BgNormal+theme.Config.FgDim+"  No field selected"+theme.Config.Reset, width, theme.Config.BgNormal)},
			width,
			height,
			theme.Config.BgNormal,
		)
	}

	lines := make([]string, 0, height)
	addLine := newConfigDetailLineAppender(width, height, &lines)
	m.renderConfigFieldSummary(addLine, field)
	m.renderConfigDetailEditor(addLine, field)

	return appendConfigPanePadding(lines, width, height, theme.Config.BgNormal)
}

func newConfigDetailLineAppender(width, height int, lines *[]string) func(string) {
	return func(text string) {
		if len(*lines) >= height {
			return
		}
		*lines = append(*lines, termtext.FillANSITextWidth(theme.Config.BgNormal+text+theme.Config.Reset, width, theme.Config.BgNormal))
	}
}

func (m Model) renderConfigFieldSummary(addLine func(string), field *config.ConfigField) {
	addLine(theme.Config.Bold + theme.Config.FgBright + "  " + field.DisplayName)
	addLine("")
	addLine(theme.Config.FgDim + "  " + field.Description)
	addLine(theme.Config.FgDim + "  path: " + field.Path)
	addLine(theme.Config.FgDim + "  type: " + field.FieldType.String())
	addLine("")
	addLine(theme.Config.FgNormal + "  current: " + theme.Config.FgBright + formatConfigValue(field.Current, field.FieldType))
	if field.Default != nil {
		addLine(theme.Config.FgDim + "  default: " + formatConfigValue(field.Default, field.FieldType))
	}
	addLine("")
}

func (m Model) renderConfigDetailEditor(addLine func(string), field *config.ConfigField) {
	switch m.configScreen.editMode {
	case editSelect:
		m.renderConfigSelectDetail(addLine, field)
	case editInput:
		addLine(theme.Config.FgCyan + "  Edit:")
		addLine("  " + theme.Config.FgBright + m.configScreen.editInput.View())
	case editSlice:
		m.renderConfigSliceDetail(addLine)
	case editStructMap:
		m.renderConfigStructMapDetail(addLine, field)
	default:
		addLine(configDetailHint(field.FieldType))
	}
}

func configDetailHint(fieldType config.ConfigFieldType) string {
	switch fieldType {
	case config.FieldTypeBool:
		return theme.Config.FgDim + "  Press Space or Enter to toggle"
	case config.FieldTypeSelect:
		return theme.Config.FgDim + "  Press Enter to select from options"
	case config.FieldTypeStringSlice:
		return theme.Config.FgDim + "  Press Enter to edit items"
	case config.FieldTypeStructMap:
		return theme.Config.FgDim + "  Press Enter to manage entries"
	default:
		return theme.Config.FgDim + "  Press Enter to edit"
	}
}

package tui

import "github.com/susugadx/xelyon-cli/internal/config"

// renderConfigDetailPane は右ペイン（詳細/エディタ）を構築する。
func (m Model) renderConfigDetailPane(width, height int) []string {
	if width <= 0 {
		return nil
	}

	field := m.configScreen.selectedField()
	if field == nil {
		return appendConfigPanePadding(
			[]string{fillANSITextWidth(cfgBgNormal+cfgFgDim+"  No field selected"+cfgReset, width, cfgBgNormal)},
			width,
			height,
			cfgBgNormal,
		)
	}

	lines := make([]string, 0, height)
	addLine := newConfigDetailLineAppender(width, height, &lines)
	m.renderConfigFieldSummary(addLine, field)
	m.renderConfigDetailEditor(addLine, field)

	return appendConfigPanePadding(lines, width, height, cfgBgNormal)
}

func newConfigDetailLineAppender(width, height int, lines *[]string) func(string) {
	return func(text string) {
		if len(*lines) >= height {
			return
		}
		*lines = append(*lines, fillANSITextWidth(cfgBgNormal+text+cfgReset, width, cfgBgNormal))
	}
}

func (m Model) renderConfigFieldSummary(addLine func(string), field *config.ConfigField) {
	addLine(cfgBold + cfgFgBright + "  " + field.DisplayName)
	addLine("")
	addLine(cfgFgDim + "  " + field.Description)
	addLine(cfgFgDim + "  path: " + field.Path)
	addLine(cfgFgDim + "  type: " + field.FieldType.String())
	addLine("")
	addLine(cfgFgNormal + "  current: " + cfgFgBright + formatConfigValue(field.Current, field.FieldType))
	if field.Default != nil {
		addLine(cfgFgDim + "  default: " + formatConfigValue(field.Default, field.FieldType))
	}
	addLine("")
}

func (m Model) renderConfigDetailEditor(addLine func(string), field *config.ConfigField) {
	switch m.configScreen.editMode {
	case editSelect:
		m.renderConfigSelectDetail(addLine, field)
	case editInput:
		addLine(cfgFgCyan + "  Edit:")
		addLine("  " + cfgFgBright + m.configScreen.editInput.View())
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
		return cfgFgDim + "  Press Space or Enter to toggle"
	case config.FieldTypeSelect:
		return cfgFgDim + "  Press Enter to select from options"
	case config.FieldTypeStringSlice:
		return cfgFgDim + "  Press Enter to edit items"
	case config.FieldTypeStructMap:
		return cfgFgDim + "  Press Enter to manage entries"
	default:
		return cfgFgDim + "  Press Enter to edit"
	}
}

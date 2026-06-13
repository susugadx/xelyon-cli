package tui

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderConfigSelectDetail(addLine func(string), field *config.ConfigField) {
	addLine(theme.Config.FgCyan + "  Select:")
	for i, opt := range field.Options {
		marker := "  ( ) "
		if i == m.configScreen.editSelect {
			marker = "  (*) "
			addLine(theme.Config.BgSelected + theme.Config.FgBright + marker + opt)
			continue
		}
		addLine(theme.Config.FgNormal + marker + opt)
	}
}

func (m Model) renderConfigSliceDetail(addLine func(string)) {
	cs := m.configScreen
	if field := cs.selectedField(); field != nil && cs.editingGuidanceFileChoices(field.Path) {
		m.renderGuidanceFileChoiceDetail(addLine, field)
		return
	}
	addLine(theme.Config.FgCyan + "  Items: (" + fmt.Sprintf("%d", len(cs.editSliceItems)) + ")")
	for i, item := range cs.editSliceItems {
		prefix := "    "
		bg := theme.Config.BgNormal
		if i == cs.editSliceIndex {
			prefix = "  > "
			bg = theme.Config.BgInactive
		}
		if cs.editSliceEditing && i == cs.editSliceIndex {
			addLine(bg + theme.Config.FgBright + "  > " + cs.editSliceInput.View())
			continue
		}
		addLine(bg + theme.Config.FgNormal + prefix + item)
	}
	if cs.editSliceAdding {
		addLine(theme.Config.FgCyan + "  + " + cs.editSliceInput.View())
	}
}

func (m Model) renderConfigStructMapDetail(addLine func(string), field *config.ConfigField) {
	if m.configScreen.editEntryActive {
		m.renderConfigStructEntryDetail(addLine)
		return
	}
	m.renderConfigStructKeyList(addLine, field)
}

func (m Model) renderConfigStructKeyList(addLine func(string), field *config.ConfigField) {
	cs := m.configScreen
	addLine(theme.Config.FgCyan + "  Keys: (" + fmt.Sprintf("%d", len(cs.editStructKeys)) + ")")
	for i, key := range cs.editStructKeys {
		prefix := "    "
		bg := theme.Config.BgNormal
		if i == cs.editStructIndex {
			prefix = "  > "
			bg = theme.Config.BgInactive
		}
		addLine(bg + theme.Config.FgNormal + prefix + key)
	}
	if cs.editStructAdding {
		addLine(theme.Config.FgCyan + "  + " + cs.editStructInput.View())
	}
	if cs.editStructIndex < 0 || cs.editStructIndex >= len(cs.editStructKeys) {
		return
	}

	key := cs.editStructKeys[cs.editStructIndex]
	summary := cs.loadEntryFields(field.Path, key)
	if len(summary) == 0 {
		return
	}
	addLine("")
	addLine(theme.Config.FgCyan + "  " + key + ":")
	for _, ef := range summary {
		addLine(theme.Config.FgDim + "    " + ef.Name + ": " + theme.Config.FgNormal + entryFieldValueStr(ef))
	}
}

func (m Model) renderConfigStructEntryDetail(addLine func(string)) {
	cs := m.configScreen
	addLine(theme.Config.FgCyan + "  " + cs.editEntryKey + ":")
	addLine("")

	for i, ef := range cs.editEntryFields {
		prefix := "    "
		bg := theme.Config.BgNormal
		if i == cs.editEntryIndex {
			prefix = "  > "
			bg = theme.Config.BgInactive
		}

		if i == cs.editEntryIndex && cs.editEntryFieldEdit == "input" {
			addLine(bg + theme.Config.FgBright + "  > " + ef.Name + ": " + cs.editInput.View())
			continue
		}
		if i == cs.editEntryIndex && cs.editEntryFieldEdit == "slice" {
			addLine(bg + theme.Config.FgBright + "  > " + ef.Name + ":")
			m.renderConfigStructEntrySliceDetail(addLine)
			continue
		}

		addLine(bg + theme.Config.FgNormal + prefix + ef.Name + ": " + theme.Config.FgDim + entryFieldValueStr(ef))
	}
}

func (m Model) renderConfigStructEntrySliceDetail(addLine func(string)) {
	cs := m.configScreen
	for i, item := range cs.editSliceItems {
		prefix := "      "
		bg := theme.Config.BgNormal
		if i == cs.editSliceIndex {
			prefix = "    > "
			bg = theme.Config.BgInactive
		}
		if cs.editSliceEditing && i == cs.editSliceIndex {
			addLine(bg + theme.Config.FgBright + "    > " + cs.editSliceInput.View())
			continue
		}
		addLine(bg + theme.Config.FgNormal + prefix + item)
	}
	if cs.editSliceAdding {
		addLine(theme.Config.FgCyan + "    + " + cs.editSliceInput.View())
	}
}

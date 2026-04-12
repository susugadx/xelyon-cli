package tui

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func (m Model) renderConfigSelectDetail(addLine func(string), field *config.ConfigField) {
	addLine(cfgFgCyan + "  Select:")
	for i, opt := range field.Options {
		marker := "  ( ) "
		if i == m.configScreen.editSelect {
			marker = "  (*) "
			addLine(cfgBgSelected + cfgFgBright + marker + opt)
			continue
		}
		addLine(cfgFgNormal + marker + opt)
	}
}

func (m Model) renderConfigSliceDetail(addLine func(string)) {
	cs := m.configScreen
	addLine(cfgFgCyan + "  Items: (" + fmt.Sprintf("%d", len(cs.editSliceItems)) + ")")
	for i, item := range cs.editSliceItems {
		prefix := "    "
		bg := cfgBgNormal
		if i == cs.editSliceIndex {
			prefix = "  > "
			bg = cfgBgInactive
		}
		if cs.editSliceEditing && i == cs.editSliceIndex {
			addLine(bg + cfgFgBright + "  > " + cs.editSliceInput.View())
			continue
		}
		addLine(bg + cfgFgNormal + prefix + item)
	}
	if cs.editSliceAdding {
		addLine(cfgFgCyan + "  + " + cs.editSliceInput.View())
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
	addLine(cfgFgCyan + "  Keys: (" + fmt.Sprintf("%d", len(cs.editStructKeys)) + ")")
	for i, key := range cs.editStructKeys {
		prefix := "    "
		bg := cfgBgNormal
		if i == cs.editStructIndex {
			prefix = "  > "
			bg = cfgBgInactive
		}
		addLine(bg + cfgFgNormal + prefix + key)
	}
	if cs.editStructAdding {
		addLine(cfgFgCyan + "  + " + cs.editStructInput.View())
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
	addLine(cfgFgCyan + "  " + key + ":")
	for _, ef := range summary {
		addLine(cfgFgDim + "    " + ef.Name + ": " + cfgFgNormal + entryFieldValueStr(ef))
	}
}

func (m Model) renderConfigStructEntryDetail(addLine func(string)) {
	cs := m.configScreen
	addLine(cfgFgCyan + "  " + cs.editEntryKey + ":")
	addLine("")

	for i, ef := range cs.editEntryFields {
		prefix := "    "
		bg := cfgBgNormal
		if i == cs.editEntryIndex {
			prefix = "  > "
			bg = cfgBgInactive
		}

		if i == cs.editEntryIndex && cs.editEntryFieldEdit == "input" {
			addLine(bg + cfgFgBright + "  > " + ef.Name + ": " + cs.editInput.View())
			continue
		}
		if i == cs.editEntryIndex && cs.editEntryFieldEdit == "slice" {
			addLine(bg + cfgFgBright + "  > " + ef.Name + ":")
			m.renderConfigStructEntrySliceDetail(addLine)
			continue
		}

		addLine(bg + cfgFgNormal + prefix + ef.Name + ": " + cfgFgDim + entryFieldValueStr(ef))
	}
}

func (m Model) renderConfigStructEntrySliceDetail(addLine func(string)) {
	cs := m.configScreen
	for i, item := range cs.editSliceItems {
		prefix := "      "
		bg := cfgBgNormal
		if i == cs.editSliceIndex {
			prefix = "    > "
			bg = cfgBgInactive
		}
		if cs.editSliceEditing && i == cs.editSliceIndex {
			addLine(bg + cfgFgBright + "    > " + cs.editSliceInput.View())
			continue
		}
		addLine(bg + cfgFgNormal + prefix + item)
	}
	if cs.editSliceAdding {
		addLine(cfgFgCyan + "    + " + cs.editSliceInput.View())
	}
}

package configscreen

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (cs *Screen) renderConfigSelectDetail(addLine func(string), field *config.ConfigField) {
	addLine(theme.Config.FgCyan + "  Select:")
	for i, opt := range field.Options {
		marker := "  ( ) "
		if i == cs.editSelect {
			marker = "  (*) "
			addLine(theme.Config.BgSelected + theme.Config.FgBright + marker + opt)
			continue
		}
		addLine(theme.Config.FgNormal + marker + opt)
	}
}

func (cs *Screen) renderConfigSliceDetail(addLine func(string)) {
	if field := cs.selectedField(); field != nil && cs.editingGuidanceFileChoices(field.Path) {
		cs.renderGuidanceFileChoiceDetail(addLine, field)
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

func (cs *Screen) renderConfigStructMapDetail(addLine func(string), field *config.ConfigField) {
	if cs.editEntryActive {
		cs.renderConfigStructEntryDetail(addLine)
		return
	}
	cs.renderConfigStructKeyList(addLine, field)
}

func (cs *Screen) renderConfigStructKeyList(addLine func(string), field *config.ConfigField) {
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

func (cs *Screen) renderConfigStructEntryDetail(addLine func(string)) {
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
			cs.renderConfigStructEntrySliceDetail(addLine)
			continue
		}

		addLine(bg + theme.Config.FgNormal + prefix + ef.Name + ": " + theme.Config.FgDim + entryFieldValueStr(ef))
	}
}

func (cs *Screen) renderConfigStructEntrySliceDetail(addLine func(string)) {
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

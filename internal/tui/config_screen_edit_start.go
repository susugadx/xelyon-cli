package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/configedit"
)

func (cs *configScreen) spaceToggle(providerConfigKey string) {
	if cs.activePane != paneField && cs.activePane != paneDetail {
		return
	}
	field := cs.selectedField()
	if field == nil || field.FieldType != config.FieldTypeBool {
		return
	}
	cs.toggleBoolField(field, providerConfigKey)
}

func (cs *configScreen) handleBrowseEnter(layout configLayout, providerConfigKey string) (configCommand, tea.Cmd) {
	switch cs.activePane {
	case paneCategory:
		cs.resetFieldSelection()
		cs.activePane = paneField
		return configCommandNone, nil
	case paneField, paneDetail:
		field := cs.selectedField()
		if field == nil {
			return configCommandNone, nil
		}
		return cs.startFieldEdit(field, layout.EditTargetPane(), providerConfigKey)
	default:
		return configCommandNone, nil
	}
}

func (cs *configScreen) startFieldEdit(field *config.ConfigField, targetPane configPane, providerConfigKey string) (configCommand, tea.Cmd) {
	switch field.FieldType {
	case config.FieldTypeBool:
		cs.toggleBoolField(field, providerConfigKey)
	case config.FieldTypeSelect:
		cs.beginSelectEdit(field, targetPane)
	case config.FieldTypeString:
		cs.beginInputEdit(targetPane, field.Current.(string))
	case config.FieldTypeInt:
		cs.beginInputEdit(targetPane, strconv.Itoa(configedit.ExtractIntCurrentValue(field.Current)))
	case config.FieldTypeFloat:
		cs.beginInputEdit(targetPane, fmt.Sprintf("%g", configedit.ExtractFloatCurrentValue(field.Current)))
	case config.FieldTypeStringSlice:
		cs.beginSliceEdit(targetPane, field)
	case config.FieldTypeStructMap:
		cs.beginStructMapEdit(targetPane, field)
	}
	return configCommandNone, nil
}

func (cs *configScreen) toggleBoolField(field *config.ConfigField, providerConfigKey string) {
	current, _ := field.Current.(bool)
	cs.applyFieldValue(field.Path, !current, providerConfigKey)
}

func (cs *configScreen) beginSelectEdit(field *config.ConfigField, targetPane configPane) {
	cs.editMode = editSelect
	cs.editSelect = 0
	current, _ := field.Current.(string)
	for i, opt := range field.Options {
		if opt == current {
			cs.editSelect = i
			break
		}
	}
	cs.activePane = targetPane
}

func (cs *configScreen) beginInputEdit(targetPane configPane, value string) {
	cs.editMode = editInput
	cs.editInput.SetValue(value)
	cs.editInput.Focus()
	cs.editInput.CursorEnd()
	cs.activePane = targetPane
}

func (cs *configScreen) beginSliceEdit(targetPane configPane, field *config.ConfigField) {
	cs.editMode = editSlice
	if slice, ok := field.Current.([]string); ok {
		cs.editSliceItems = make([]string, len(slice))
		copy(cs.editSliceItems, slice)
	} else {
		cs.editSliceItems = nil
	}
	cs.editSliceIndex = 0
	cs.editSliceAdding = false
	cs.editSliceEditing = false
	cs.editGuidanceChoices = guidanceFileChoicesForField(field.Path, cs.editSliceItems)
	cs.editGuidanceIndex = 0
	cs.activePane = targetPane
}

func (cs *configScreen) beginStructMapEdit(targetPane configPane, field *config.ConfigField) {
	cs.editMode = editStructMap
	cs.editStructKeys = cs.structMapKeys(field.Path)
	cs.editStructIndex = 0
	cs.editStructAdding = false
	cs.activePane = targetPane
}

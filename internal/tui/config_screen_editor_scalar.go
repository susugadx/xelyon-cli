package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/configedit"
)

func (cs *configScreen) handleEditKey(msg tea.KeyMsg, providerConfigKey string) (configCommand, tea.Cmd) {
	switch cs.editMode {
	case editSelect:
		return cs.handleSelectEdit(msg, providerConfigKey)
	case editInput:
		return cs.handleInputEdit(msg, providerConfigKey)
	case editSlice:
		return cs.handleSliceEdit(msg)
	case editStructMap:
		return cs.handleStructMapEdit(msg)
	}
	return configCommandNone, nil
}

func (cs *configScreen) handleSelectEdit(msg tea.KeyMsg, providerConfigKey string) (configCommand, tea.Cmd) {
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return configCommandNone, nil
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return configCommandNone, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSelect > 0 {
			cs.editSelect--
		}
		return configCommandNone, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSelect < len(field.Options)-1 {
			cs.editSelect++
		}
		return configCommandNone, nil

	case isEnterKey(msg):
		if cs.editSelect >= 0 && cs.editSelect < len(field.Options) {
			if !cs.applyFieldValue(field.Path, field.Options[cs.editSelect], providerConfigKey) {
				return configCommandNone, nil
			}
		}
		cs.editMode = editNone
		return configCommandNone, nil
	}
	return configCommandNone, nil
}

func (cs *configScreen) handleInputEdit(msg tea.KeyMsg, providerConfigKey string) (configCommand, tea.Cmd) {
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return configCommandNone, nil
	}

	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		cs.editInput.Blur()
		return configCommandNone, nil

	case isEnterKey(msg):
		raw := cs.editInput.Value()
		var newVal interface{}
		changed := false

		switch field.FieldType {
		case config.FieldTypeString:
			newVal = raw
			changed = true
		case config.FieldTypeInt:
			v, inputChanged, valid := configedit.ParseIntInput(raw, configedit.ExtractIntCurrentValue(field.Current))
			if !valid || !inputChanged {
				return configCommandNone, nil
			}
			newVal = v
			changed = inputChanged
		case config.FieldTypeFloat:
			v, inputChanged, status := configedit.ParseFloatInput(raw, configedit.ExtractFloatCurrentValue(field.Current), field.Path)
			if status != configedit.FloatInputValid || !inputChanged {
				return configCommandNone, nil
			}
			newVal = v
			changed = inputChanged
		}

		if changed && newVal != nil {
			if !cs.applyFieldValue(field.Path, newVal, providerConfigKey) {
				return configCommandNone, nil
			}
		}
		cs.editMode = editNone
		cs.editInput.Blur()
		return configCommandNone, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return configCommandNone, cmd
	}
}

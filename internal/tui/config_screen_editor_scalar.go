package tui

import (
	"math"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
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

		switch field.FieldType {
		case config.FieldTypeString:
			newVal = raw
		case config.FieldTypeInt:
			v, err := strconv.Atoi(raw)
			if err != nil {
				return configCommandNone, nil
			}
			newVal = v
		case config.FieldTypeFloat:
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return configCommandNone, nil
			}
			if field.Path == "project_map.context_ratio" &&
				(v < config.ProjectMapContextRatioMin || v > config.ProjectMapContextRatioMax) {
				return configCommandNone, nil
			}
			newVal = v
		}

		if newVal != nil {
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

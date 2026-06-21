package uiconfig

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/configedit"
)

func (m *ConfigMenu) editBool(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(bool)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %v\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (y/n, or Enter to keep): ")

	input := strings.TrimSpace(strings.ToLower(readLineWithIO(&promptIO)))
	value, changed, valid := configedit.ParseBoolInput(input, current)
	if !valid {
		_, _ = fmt.Fprintf(out, "%sInvalid input, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
	if !changed {
		return current, false, nil
	}
	return value, true, nil
}

func (m *ConfigMenu) editInt(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current := configedit.ExtractIntCurrentValue(field.Current)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %d\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	value, changed, valid := configedit.ParseIntInput(input, current)
	if !valid {
		_, _ = fmt.Fprintf(out, "%sInvalid number, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
	if !changed {
		return current, false, nil
	}
	return value, true, nil
}

func (m *ConfigMenu) editFloat(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current := configedit.ExtractFloatCurrentValue(field.Current)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %g\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	value, changed, status := configedit.ParseFloatInput(input, current, field.Path)
	switch status {
	case configedit.FloatInputKeep:
		return current, false, nil
	case configedit.FloatInputInvalid:
		_, _ = fmt.Fprintf(out, "%sInvalid number, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	case configedit.FloatInputOutOfRange:
		_, _ = fmt.Fprintf(out, "%sValue must be between %.2f and %.2f, keeping current value%s\n",
			colorDim, config.ProjectMapContextRatioMin, config.ProjectMapContextRatioMax, colorReset)
		return current, false, nil
	case configedit.FloatInputValid:
		if !changed {
			return current, false, nil
		}
		return value, true, nil
	default:
		return current, false, nil
	}
}

func (m *ConfigMenu) editString(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %s\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := readLineWithIO(&promptIO)
	value, changed := configedit.ParseStringInput(input, current)
	if !changed {
		return current, false, nil
	}

	return value, true, nil
}

func (m *ConfigMenu) editSelect(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %s\n", current)
	_, _ = fmt.Fprintln(out, "Available options:")

	for i, opt := range field.Options {
		marker := "  "
		if opt == current {
			marker = "▶ "
		}
		_, _ = fmt.Fprintf(out, "  %s%d. %s\n", marker, i+1, opt)
	}

	_, _ = fmt.Fprintf(out, "\nEnter number (1-%d) or Enter to keep: ", len(field.Options))

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	value, changed, valid := configedit.ParseSelectInput(input, current, field.Options)
	if !valid {
		_, _ = fmt.Fprintf(out, "%sInvalid selection, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
	if !changed {
		return current, false, nil
	}
	return value, true, nil
}

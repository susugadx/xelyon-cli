package ui

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// EditField はフィールドを編集する。
func (m *ConfigMenu) EditField(field *config.ConfigField) (interface{}, bool, error) {
	ctx := newConfigPromptContext(m.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	m.renderFieldEditSummary(out, field)

	switch field.FieldType {
	case config.FieldTypeBool:
		return m.editBool(promptIO, field)
	case config.FieldTypeInt:
		return m.editInt(promptIO, field)
	case config.FieldTypeFloat:
		return m.editFloat(promptIO, field)
	case config.FieldTypeString:
		return m.editString(promptIO, field)
	case config.FieldTypeSelect:
		return m.editSelect(promptIO, field)
	case config.FieldTypeStringSlice:
		return m.editStringSlice(field)
	case config.FieldTypeStringMap:
		return m.editStringMap(field)
	case config.FieldTypeStructMap:
		return m.editStructMap(field)
	default:
		_, _ = fmt.Fprintf(out, "%sUnsupported field type%s\n", colorDim, colorReset)
		return nil, false, nil
	}
}

func (m *ConfigMenu) renderFieldEditSummary(out io.Writer, field *config.ConfigField) {
	_, _ = fmt.Fprintf(out, "\n%s%s%s\n", colorCyan, field.Description, colorReset)
	_, _ = fmt.Fprintf(out, "Path: %s\n", field.Path)
	_, _ = fmt.Fprintf(out, "Type: %s\n", field.FieldType.String())
	_, _ = fmt.Fprintf(out, "Current: %v\n", formatValue(field.Current))
	if field.Default != nil {
		_, _ = fmt.Fprintf(out, "Default: %v\n", formatValue(field.Default))
	}
	_, _ = fmt.Fprintln(out)
}

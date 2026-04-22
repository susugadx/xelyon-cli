package ui

import "github.com/susugadx/xelyon-cli/internal/config"

func (m *ConfigMenu) editStringSlice(field *config.ConfigField) (interface{}, bool, error) {
	current := []string{}
	if slice, ok := field.Current.([]string); ok {
		current = slice
	}

	editor := NewStringSliceEditorWithRuntime(field.Path, current, m.Runtime)
	result, changed, err := editor.Run()
	if err != nil {
		return nil, false, err
	}

	return result, changed, nil
}

func (m *ConfigMenu) editStringMap(field *config.ConfigField) (interface{}, bool, error) {
	current := map[string]string{}
	if mp, ok := field.Current.(map[string]string); ok {
		current = mp
	}

	editor := NewStringMapEditorWithRuntime(field.Path, current, m.Runtime)
	result, changed, err := editor.Run()
	if err != nil {
		return nil, false, err
	}

	return result, changed, nil
}

func (m *ConfigMenu) editStructMap(field *config.ConfigField) (interface{}, bool, error) {
	editor := NewStructMapEditorWithRuntime(field.Path, field.FieldType, m.Runtime)
	changed, err := editor.Run(m.Config)
	if err != nil {
		return nil, false, err
	}

	// StructMapEditorは直接Configを編集するので、変更されたかどうかのみ返す
	return nil, changed, nil
}

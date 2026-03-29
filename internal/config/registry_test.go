package config

import (
	"testing"
)

func TestBuildConfigRegistry(t *testing.T) {
	cfg := DefaultConfig()
	categories := BuildConfigRegistry(cfg)

	// カテゴリ数の確認
	if len(categories) == 0 {
		t.Error("BuildConfigRegistry returned empty categories")
	}

	// 各カテゴリにフィールドがあることを確認
	for _, cat := range categories {
		if cat.Name == "" {
			t.Error("Category has empty name")
		}
		if cat.DisplayName == "" {
			t.Errorf("Category %s has empty display name", cat.Name)
		}
		if cat.Icon == "" {
			t.Errorf("Category %s has empty icon", cat.Name)
		}
	}

	// provider カテゴリの確認
	var providerCat *ConfigCategory
	for i := range categories {
		if categories[i].Name == "provider" {
			providerCat = &categories[i]
			break
		}
	}

	if providerCat == nil {
		t.Error("Provider category not found")
	} else {
		// provider カテゴリには default_provider, default_model, provider_models が含まれる
		fieldPaths := make(map[string]bool)
		for _, field := range providerCat.Fields {
			fieldPaths[field.Path] = true
		}

		expectedFields := []string{"default_provider", "default_model", "provider_models"}
		for _, expected := range expectedFields {
			if !fieldPaths[expected] {
				t.Errorf("Provider category missing field: %s", expected)
			}
		}
	}
}

func TestGetFieldValue(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		path     string
		wantType string
	}{
		{"default_provider", "default_provider", "string"},
		{"default_model", "default_model", "string"},
		{"compression.auto_compress", "compression.auto_compress", "bool"},
		{"api_retry.count", "api_retry.count", "int"},
		{"thinking.enabled", "thinking.enabled", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := GetFieldValue(cfg, tt.path)
			if err != nil {
				t.Errorf("GetFieldValue(%s) error = %v", tt.path, err)
				return
			}
			if val == nil {
				t.Errorf("GetFieldValue(%s) returned nil", tt.path)
			}
		})
	}
}

func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   interface{}
		wantErr bool
	}{
		{"set string", "default_model", "new-model", false},
		{"set bool", "thinking.enabled", true, false},
		{"set int", "api_retry.count", 5, false},
		{"invalid path", "nonexistent.field", "value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			err := SetFieldValue(cfg, tt.path, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetFieldValue(%s) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 設定された値を確認
				gotVal, _ := GetFieldValue(cfg, tt.path)
				if gotVal != tt.value {
					t.Errorf("SetFieldValue(%s) value = %v, want %v", tt.path, gotVal, tt.value)
				}
			}
		})
	}
}

func TestFieldTypeMap(t *testing.T) {
	// FieldTypeMap に重要なフィールドが含まれていることを確認
	requiredFields := map[string]ConfigFieldType{
		"default_provider":                    FieldTypeSelect,
		"default_model":                       FieldTypeString,
		"compression.auto_compress":           FieldTypeBool,
		"compression.clear_tool_uses":         FieldTypeBool,
		"compression.clear_tool_inputs":       FieldTypeBool,
		"compression.compaction_trigger":      FieldTypeInt,
		"api_retry.count":                     FieldTypeInt,
		"bash.safety_level":                   FieldTypeSelect,
		"bash.safe_commands":                  FieldTypeStringSlice,
		"command_aliases":                     FieldTypeStringMap,
		"output.assistant_updates":            FieldTypeSelect,
		"plan_mode.clear_context_on_approval": FieldTypeBool,
		"project_map.context_ratio":           FieldTypeFloat,
		"provider_models":                     FieldTypeStructMap,
		"sub_agent.max_concurrent":            FieldTypeInt,
	}

	for path, expectedType := range requiredFields {
		actualType, ok := FieldTypeMap[path]
		if !ok {
			t.Errorf("FieldTypeMap missing field: %s", path)
			continue
		}
		if actualType != expectedType {
			t.Errorf("FieldTypeMap[%s] = %v, want %v", path, actualType, expectedType)
		}
	}
}

func TestSelectOptions(t *testing.T) {
	// SelectOptions に重要なフィールドの選択肢が含まれていることを確認
	tests := []struct {
		path    string
		minOpts int
	}{
		{"default_provider", 6},  // deepseek, claude, openai, gemini, groq, ollama
		{"bash.safety_level", 3}, // strict, moderate, permissive
		{"thinking.level", 4},    // low, medium, high, xhigh
		{"output.assistant_updates", 4},
	}

	for _, tt := range tests {
		opts, ok := SelectOptions[tt.path]
		if !ok {
			t.Errorf("SelectOptions missing field: %s", tt.path)
			continue
		}
		if len(opts) < tt.minOpts {
			t.Errorf("SelectOptions[%s] has %d options, want at least %d", tt.path, len(opts), tt.minOpts)
		}
	}
}

func TestFieldDescriptions(t *testing.T) {
	// FieldDescriptions に説明が含まれていることを確認
	requiredFields := []string{
		"default_provider",
		"compression.auto_compress",
		"compression.clear_tool_uses",
		"compression.clear_tool_inputs",
		"api_retry.count",
		"bash.safety_level",
		"output.assistant_updates",
		"plan_mode.clear_context_on_approval",
		"project_map.context_ratio",
		"sub_agent.default_model",
	}

	for _, path := range requiredFields {
		desc, ok := FieldDescriptions[path]
		if !ok {
			t.Errorf("FieldDescriptions missing field: %s", path)
			continue
		}
		if desc == "" {
			t.Errorf("FieldDescriptions[%s] is empty", path)
		}
	}
}

package config

import (
	"os"
	"path/filepath"
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
		{"compression.enabled", "compression.enabled", "bool"},
		{"compression.keep_recent", "compression.keep_recent", "int"},
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
		{"set int", "compression.keep_recent", 5, false},
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

func TestGetFieldValue_ProviderModelsUsesRawEntriesForLoadedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: deepseek-chat
provider_models:
  anthropic:
    default_model: anthropic-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	val, err := GetFieldValue(cfg, "provider_models")
	if err != nil {
		t.Fatalf("GetFieldValue(provider_models) error = %v", err)
	}

	providerModels, ok := val.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("GetFieldValue(provider_models) returned %T, want map[string]ProviderModelConfig", val)
	}
	if len(providerModels) != 1 {
		t.Fatalf("len(provider_models) = %d, want 1 raw entry", len(providerModels))
	}
	if _, ok := providerModels["anthropic"]; !ok {
		t.Fatal("provider_models should contain anthropic raw entry")
	}
	if _, ok := providerModels["claude"]; ok {
		t.Fatal("provider_models should not expose effective default claude entry in editor view")
	}
}

func TestGetFieldValue_ProviderModelsIsEmptyWhenLoadedConfigHasNoSection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: gpt-5.4
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	val, err := GetFieldValue(cfg, "provider_models")
	if err != nil {
		t.Fatalf("GetFieldValue(provider_models) error = %v", err)
	}

	providerModels, ok := val.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("GetFieldValue(provider_models) returned %T, want map[string]ProviderModelConfig", val)
	}
	if len(providerModels) != 0 {
		t.Fatalf("len(provider_models) = %d, want 0 when section is absent", len(providerModels))
	}
}

func TestSetFieldValue_ProviderModelsUpdatesRawSaveState(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "old-model"},
	})
	cfg.refreshEffectiveProviderModels()

	newProviderModels := map[string]ProviderModelConfig{
		"openai": {DefaultModel: "new-model"},
	}
	if err := SetFieldValue(cfg, "provider_models", newProviderModels); err != nil {
		t.Fatalf("SetFieldValue(provider_models) error = %v", err)
	}

	saved := cfg.ProviderModelsForSave()
	if got := saved["openai"].DefaultModel; got != "new-model" {
		t.Fatalf("ProviderModelsForSave()[openai].DefaultModel = %q, want %q", got, "new-model")
	}
}

func TestSetFieldValue_ProviderModelsPreservesSeparateClaudeAnthropicAliasEntries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"

	newProviderModels := map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude": {
			DefaultModel:     "claude-custom",
			AnthropicVersion: "2099-01-01",
		},
	}
	if err := SetFieldValue(cfg, "provider_models", newProviderModels); err != nil {
		t.Fatalf("SetFieldValue(provider_models) error = %v", err)
	}

	saved := cfg.ProviderModelsForSave()
	if len(saved) != 2 {
		t.Fatalf("len(ProviderModelsForSave()) = %d, want 2", len(saved))
	}

	pm, ok := saved["anthropic"]
	if !ok {
		t.Fatal("ProviderModelsForSave()['anthropic'] missing")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.AnthropicVersion != "" {
		t.Fatalf("ProviderModelsForSave()['anthropic'].AnthropicVersion = %q, want empty", pm.AnthropicVersion)
	}
	if claudePM, ok := saved["claude"]; !ok {
		t.Fatal("ProviderModelsForSave()['claude'] missing")
	} else if claudePM.DefaultModel != "claude-custom" {
		t.Fatalf("ProviderModelsForSave()['claude'].DefaultModel = %q, want %q", claudePM.DefaultModel, "claude-custom")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
}

func TestBuildConfigRegistry_ProviderModelsDefaultResetsToAbsentSection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "custom-global-model"

	categories := BuildConfigRegistry(cfg)
	var providerModelsField *ConfigField
	for i := range categories {
		for j := range categories[i].Fields {
			if categories[i].Fields[j].Path == "provider_models" {
				providerModelsField = &categories[i].Fields[j]
				break
			}
		}
	}

	if providerModelsField == nil {
		t.Fatal("provider_models field not found")
	}

	defaultVal, ok := providerModelsField.Default.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("provider_models default has type %T, want map[string]ProviderModelConfig", providerModelsField.Default)
	}
	if defaultVal != nil {
		t.Fatalf("provider_models default = %#v, want nil map to represent absent section", defaultVal)
	}

	if err := SetFieldValue(cfg, "provider_models", defaultVal); err != nil {
		t.Fatalf("SetFieldValue(provider_models, default) error = %v", err)
	}
	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after reset to default", got)
	}
	if got := cfg.GetSelectedModelForProvider("openai"); got != "custom-global-model" {
		t.Fatalf("GetSelectedModelForProvider(openai) = %q, want %q", got, "custom-global-model")
	}
}

func TestSetFieldValue_ProviderModelsNilResetRemovesExplicitSection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"openai": {DefaultModel: "gpt-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if err := SetFieldValue(cfg, "provider_models", map[string]ProviderModelConfig(nil)); err != nil {
		t.Fatalf("SetFieldValue(provider_models, nil) error = %v", err)
	}

	if got := cfg.ProviderModelsForSave(); got != nil {
		t.Fatalf("ProviderModelsForSave() = %#v, want nil after reset", got)
	}
	if got := cfg.providerModelSectionState(); got != providerModelSectionStateAbsent {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateAbsent)
	}
}

func TestFieldTypeMap(t *testing.T) {
	// FieldTypeMap に重要なフィールドが含まれていることを確認
	requiredFields := map[string]ConfigFieldType{
		"default_provider":            FieldTypeSelect,
		"default_model":               FieldTypeString,
		"general.ui_language":         FieldTypeSelect,
		"compression.enabled":         FieldTypeBool,
		"compression.trigger_percent": FieldTypeInt,
		"compression.keep_recent":     FieldTypeInt,
		"execution.mode":              FieldTypeSelect,
		"output.assistant_updates":    FieldTypeSelect,
		"project_map.context_ratio":   FieldTypeFloat,
		"provider_models":             FieldTypeStructMap,
		"sub_agent.max_concurrent":    FieldTypeInt,
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
		{"default_provider", 6},    // deepseek, claude, openai, gemini, groq, ollama
		{"web_search.provider", 4}, // openai, gemini, claude, anthropic
		{"execution.mode", 3},      // balanced, trusted, full_auto
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

func TestSelectOptions_WebSearchProviderIncludesAnthropicAlias(t *testing.T) {
	opts, ok := SelectOptions["web_search.provider"]
	if !ok {
		t.Fatal("SelectOptions missing field: web_search.provider")
	}

	required := []string{"openai", "gemini", "claude", "anthropic"}
	for _, want := range required {
		found := false
		for _, opt := range opts {
			if opt == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("SelectOptions[web_search.provider] = %v, want to contain %q", opts, want)
		}
	}
}

func TestFieldDescriptions(t *testing.T) {
	// FieldDescriptions に説明が含まれていることを確認
	requiredFields := []string{
		"default_provider",
		"general.ui_language",
		"compression.enabled",
		"compression.trigger_percent",
		"compression.keep_recent",
		"execution.mode",
		"output.assistant_updates",
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

func TestInternalSectionsNotInRegistry(t *testing.T) {
	// thinking / openai は user-facing config から削除済み
	// make gen-all で誤って再導入されないことを保証する
	internalFields := []string{
		"thinking.enabled",
		"thinking.level",
		"openai.responses_api_models",
	}

	for _, path := range internalFields {
		if _, ok := FieldTypeMap[path]; ok {
			t.Errorf("FieldTypeMap should not contain internal field: %s", path)
		}
		if _, ok := FieldDescriptions[path]; ok {
			t.Errorf("FieldDescriptions should not contain internal field: %s", path)
		}
	}

	internalCategories := []string{"thinking", "openai"}
	for _, cat := range internalCategories {
		for _, def := range CategoryDefinitions {
			if def.Name == cat {
				t.Errorf("CategoryDefinitions should not contain internal category: %s", cat)
			}
		}
		if _, ok := SelectOptions["thinking.level"]; ok {
			t.Errorf("SelectOptions should not contain internal field: thinking.level")
		}
	}
}

package config

import "testing"

func TestFieldTypeMap(t *testing.T) {
	// FieldTypeMap に重要なフィールドが含まれていることを確認
	requiredFields := map[string]ConfigFieldType{
		"default_provider":                  FieldTypeSelect,
		"default_model":                     FieldTypeString,
		"general.ui_language":               FieldTypeSelect,
		"compression.enabled":               FieldTypeBool,
		"compression.trigger_percent":       FieldTypeInt,
		"compression.keep_recent":           FieldTypeInt,
		"execution.mode":                    FieldTypeSelect,
		"output.assistant_updates":          FieldTypeSelect,
		"project_map.context_ratio":         FieldTypeFloat,
		"agent_instructions.project.mode":   FieldTypeSelect,
		"agent_instructions.max_file_bytes": FieldTypeInt,
		"provider_models":                   FieldTypeStructMap,
		"sub_agent.max_concurrent":          FieldTypeInt,
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
		{"agent_instructions.project.mode", 3},
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
		"responses.store",
		"responses.persist_response_id",
		"responses.server_compaction.enabled",
	}

	for _, path := range internalFields {
		if _, ok := FieldTypeMap[path]; ok {
			t.Errorf("FieldTypeMap should not contain internal field: %s", path)
		}
		if _, ok := FieldDescriptions[path]; ok {
			t.Errorf("FieldDescriptions should not contain internal field: %s", path)
		}
	}

	internalCategories := []string{"thinking", "openai", "responses"}
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

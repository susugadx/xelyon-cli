package config

import "testing"

func TestFieldTypeMap(t *testing.T) {
	// FieldTypeMap に重要なフィールドが含まれていることを確認
	requiredFields := map[string]ConfigFieldType{
		"default_provider":                                                                 FieldTypeSelect,
		"default_model":                                                                    FieldTypeString,
		"general.ui_language":                                                              FieldTypeSelect,
		"compression.enabled":                                                              FieldTypeBool,
		"compression.trigger_percent":                                                      FieldTypeInt,
		"compression.keep_recent":                                                          FieldTypeInt,
		"provider_history_reduction.mode":                                                  FieldTypeSelect,
		"provider_history_reduction.rehydrate_context":                                     FieldTypeBool,
		"provider_history_reduction.raw_output_artifacts.mode":                             FieldTypeSelect,
		"provider_history_reduction.raw_output_artifacts.root":                             FieldTypeString,
		"provider_history_reduction.raw_output_artifacts.max_artifact_bytes":               FieldTypeInt,
		"provider_history_reduction.raw_output_artifacts.session_quota_bytes":              FieldTypeInt,
		"provider_history_reduction.raw_output_artifacts.chunk_bytes":                      FieldTypeInt,
		"provider_history_reduction.raw_output_artifacts.active_context_budget_tokens":     FieldTypeInt,
		"provider_history_reduction.raw_output_artifacts.active_context_budget_max_tokens": FieldTypeInt,
		"provider_history_reduction.raw_output_artifacts.retention":                        FieldTypeSelect,
		"execution.mode":                                                                   FieldTypeSelect,
		"output.assistant_updates":                                                         FieldTypeSelect,
		"project_map.context_ratio":                                                        FieldTypeFloat,
		"agent_instructions.project.mode":                                                  FieldTypeSelect,
		"agent_instructions.max_file_bytes":                                                FieldTypeInt,
		"provider_models":                                                                  FieldTypeStructMap,
		"review.provider":                                                                  FieldTypeSelect,
		"review.model":                                                                     FieldTypeString,
		"review.thinking.mode":                                                             FieldTypeSelect,
		"review.thinking.level":                                                            FieldTypeSelect,
		"gemini.service_tier":                                                              FieldTypeSelect,
		"sub_agent.max_concurrent":                                                         FieldTypeInt,
		"mcp.surface_budget.max_tools":                                                     FieldTypeInt,
		"mcp.surface_budget.estimated_tokens":                                              FieldTypeInt,
		"mcp.surface_budget.max_schema_bytes_per_tool":                                     FieldTypeInt,
		"skills.router.enabled":                                                            FieldTypeBool,
		"skills.router.activation":                                                         FieldTypeSelect,
		"skills.router.usage_ledger":                                                       FieldTypeBool,
		"skills.router.usage_retention_days":                                               FieldTypeInt,
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
		{"provider_history_reduction.mode", 3},
		{"provider_history_reduction.raw_output_artifacts.mode", 3},
		{"provider_history_reduction.raw_output_artifacts.retention", 1},
		{"agent_instructions.project.mode", 3},
		{"output.assistant_updates", 4},
		{"review.provider", 11}, // empty + display providers
		{"review.thinking.mode", 3},
		{"review.thinking.level", 5},
		{"gemini.service_tier", 3},
		{"skills.router.activation", 2},
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

func TestSelectOptionsProviderHistoryReductionModeDoesNotExposeAuto(t *testing.T) {
	opts, ok := SelectOptions["provider_history_reduction.mode"]
	if !ok {
		t.Fatal("SelectOptions missing field: provider_history_reduction.mode")
	}
	want := []string{"off", "dry_run", "apply"}
	if len(opts) != len(want) {
		t.Fatalf("SelectOptions[provider_history_reduction.mode] = %v, want %v", opts, want)
	}
	for i := range want {
		if opts[i] != want[i] {
			t.Fatalf("SelectOptions[provider_history_reduction.mode] = %v, want %v", opts, want)
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
		"provider_history_reduction.mode",
		"provider_history_reduction.rehydrate_context",
		"provider_history_reduction.raw_output_artifacts.mode",
		"provider_history_reduction.raw_output_artifacts.root",
		"provider_history_reduction.raw_output_artifacts.active_context_budget_tokens",
		"execution.mode",
		"output.assistant_updates",
		"project_map.context_ratio",
		"review.provider",
		"review.model",
		"review.thinking.mode",
		"review.thinking.level",
		"gemini.service_tier",
		"sub_agent.default_model",
		"mcp.surface_budget.max_tools",
		"mcp.surface_budget.estimated_tokens",
		"mcp.surface_budget.max_schema_bytes_per_tool",
		"skills.router.enabled",
		"skills.router.activation",
		"skills.router.usage_ledger",
		"skills.router.usage_retention_days",
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
		"responses.server_compaction.compact_threshold",
		"responses.server_compaction.local_fallback",
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

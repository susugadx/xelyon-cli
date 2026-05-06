package config

import "testing"

func TestIsResponsesAPIModel(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-5.2-codex", true},
		{"gpt-5.3-codex", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.1-codex-max", true},
		{"gpt-5-codex", true},
		{"gpt-5.2", true},
		{"gpt-5.2-pro", true},
		{"gpt-5.1", true},
		{"gpt-5", true},
		{"gpt-5-mini", true},
		{"gpt-5-nano", true},
		{"gpt-5.3-codex-spark", true},
		{"gpt-5.1-codex-mini", true},
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"gpt-4o-audio-preview", true},
		{"o1", true},
		{"o1-mini", true},
		{"o1-pro", true},
		{"o3", true},
		{"o3-mini", true},
		{"o3-pro", true},
		{"o4-mini", true},
		{"gpt-4-turbo", false},
		{"gpt-4", false},
		{"gpt-3.5-turbo", false},
		{"openai-custom", false},
		{"unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := cfg.IsResponsesAPIModel(tt.model)
			if got != tt.expected {
				t.Errorf("IsResponsesAPIModel(%q) = %v, want %v", tt.model, got, tt.expected)
			}
		})
	}
}

func TestIsResponsesAPIModel_CustomModels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "custom-codex-model")

	if !cfg.IsResponsesAPIModel("custom-codex-model") {
		t.Error("IsResponsesAPIModel() should return true for custom model")
	}
}

func TestIsProviderResponsesAPIRequest_AzureDeploymentUsesResponsesRoute(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.IsProviderResponsesAPIRequest("azure", "azure", "azure-gpt-5.4") {
		t.Fatal("IsProviderResponsesAPIRequest(azure deployment) = false, want true")
	}
	if cfg.IsProviderResponsesAPIRequest("openai", "openai", "corp-gpt-deployment") {
		t.Fatal("IsProviderResponsesAPIRequest(openai custom deployment) = true, want false without catalog_model")
	}
}

func TestResponsesConfigDefaultsAndOverrides(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ResponsesStoreEnabled() {
		t.Fatal("ResponsesStoreEnabled() = false, want default true")
	}
	if !cfg.ResponsesPersistResponseIDEnabled() {
		t.Fatal("ResponsesPersistResponseIDEnabled() = false, want default true")
	}
	if !cfg.ResponsesServerCompactionEnabled() {
		t.Fatal("ResponsesServerCompactionEnabled() = false, want default true")
	}
	if cfg.ResponsesServerCompactionCompactThreshold() != 0 {
		t.Fatalf("ResponsesServerCompactionCompactThreshold() = %d, want default 0(auto)", cfg.ResponsesServerCompactionCompactThreshold())
	}
	if !cfg.ResponsesServerCompactionLocalFallbackEnabled() {
		t.Fatal("ResponsesServerCompactionLocalFallbackEnabled() = false, want default true")
	}

	yamlData := []byte(`
responses:
  store: false
  persist_response_id: true
`)
	loaded, err := loadConfigFromData(yamlData)
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}
	if loaded.ResponsesStoreEnabled() {
		t.Fatal("ResponsesStoreEnabled() = true, want false from YAML")
	}
	if loaded.ResponsesPersistResponseIDEnabled() {
		t.Fatal("ResponsesPersistResponseIDEnabled() = true, want false when store is disabled")
	}
	if loaded.ResponsesServerCompactionEnabled() {
		t.Fatal("ResponsesServerCompactionEnabled() = true, want false when store is disabled")
	}

	yamlData = []byte(`
responses:
  persist_response_id: false
  server_compaction:
    enabled: false
    compact_threshold: 8000
    local_fallback: false
`)
	loaded, err = loadConfigFromData(yamlData)
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}
	if !loaded.ResponsesStoreEnabled() {
		t.Fatal("ResponsesStoreEnabled() = false, want omitted store to keep default true")
	}
	if loaded.ResponsesPersistResponseIDEnabled() {
		t.Fatal("ResponsesPersistResponseIDEnabled() = true, want false from YAML")
	}
	if loaded.ResponsesServerCompactionEnabled() {
		t.Fatal("ResponsesServerCompactionEnabled() = true, want false from YAML")
	}
	if loaded.ResponsesServerCompactionCompactThreshold() != 8000 {
		t.Fatalf("ResponsesServerCompactionCompactThreshold() = %d, want 8000 from YAML", loaded.ResponsesServerCompactionCompactThreshold())
	}
	if loaded.ResponsesServerCompactionLocalFallbackEnabled() {
		t.Fatal("ResponsesServerCompactionLocalFallbackEnabled() = true, want false from YAML")
	}
}

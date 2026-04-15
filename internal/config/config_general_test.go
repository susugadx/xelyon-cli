package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "deepseek-chat" {
		t.Errorf("DefaultModel = %v, want deepseek-chat", cfg.DefaultModel)
	}
	if cfg.ProviderModels == nil {
		t.Fatal("ProviderModels is nil")
	}

	expectedProviders := []string{"deepseek", "openai", "gemini", "claude", "ollama", "groq", "openrouter", "bedrock"}
	for _, provider := range expectedProviders {
		if _, ok := cfg.ProviderModels[provider]; !ok {
			t.Errorf("ProviderModels missing provider: %s", provider)
		}
	}

	if cfg.Compression.Enabled != true {
		t.Error("Compression.Enabled should default to true")
	}
	if cfg.Compression.ThresholdTokens != 0 {
		t.Errorf("Compression.ThresholdTokens = %d, want 0 (percentage-based)", cfg.Compression.ThresholdTokens)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent = %d, want 80", cfg.Compression.TriggerPercent)
	}
	if cfg.Compression.TokenThreshold != 0 {
		t.Errorf("Compression.TokenThreshold = %d, want 0", cfg.Compression.TokenThreshold)
	}
	if cfg.Compression.Model != "" {
		t.Errorf("Compression.Model = %q, want empty string", cfg.Compression.Model)
	}
	if cfg.Compression.KeepRecent != 20 {
		t.Errorf("Compression.KeepRecent = %d, want 20", cfg.Compression.KeepRecent)
	}
	if cfg.Compression.PreferCompactAPI != true {
		t.Error("Compression.PreferCompactAPI should default to true")
	}
	if cfg.Compression.ClaudeCompaction != true {
		t.Error("Compression.ClaudeCompaction should default to true")
	}
	if cfg.Compression.CompactionTrigger != 150000 {
		t.Errorf("Compression.CompactionTrigger = %d, want 150000", cfg.Compression.CompactionTrigger)
	}
	if cfg.Compression.ClearToolUses != true {
		t.Error("Compression.ClearToolUses should default to true")
	}
	if cfg.Compression.ClearToolUsesTrigger != 80000 {
		t.Errorf("Compression.ClearToolUsesTrigger = %d, want 80000", cfg.Compression.ClearToolUsesTrigger)
	}
	if cfg.Compression.ClearToolInputs {
		t.Error("Compression.ClearToolInputs should default to false")
	}
	if cfg.LoopDetection.Threshold != 3 {
		t.Errorf("LoopDetection.Threshold = %d, want 3", cfg.LoopDetection.Threshold)
	}
	if cfg.APIRetry.Count != 3 {
		t.Errorf("APIRetry.Count = %d, want 3", cfg.APIRetry.Count)
	}
	if cfg.APIRetry.InitialDelay != 1 {
		t.Errorf("APIRetry.InitialDelay = %d, want 1", cfg.APIRetry.InitialDelay)
	}
	if cfg.APIRetry.MaxDelay != 30 {
		t.Errorf("APIRetry.MaxDelay = %d, want 30", cfg.APIRetry.MaxDelay)
	}
	if cfg.Diff.ContextLines != 10 {
		t.Errorf("Diff.ContextLines = %d, want 10", cfg.Diff.ContextLines)
	}
	if cfg.General.ToolLoopLimit != 0 {
		t.Errorf("General.ToolLoopLimit = %d, want 0", cfg.General.ToolLoopLimit)
	}
	if !cfg.ProjectMap.Enabled {
		t.Error("ProjectMap.Enabled should default to true")
	}
	if cfg.ProjectMap.ContextRatio != ProjectMapContextRatioDefault {
		t.Errorf("ProjectMap.ContextRatio = %f, want %f", cfg.ProjectMap.ContextRatio, ProjectMapContextRatioDefault)
	}
	if !cfg.SubAgent.Enabled {
		t.Error("SubAgent.Enabled should default to true")
	}
	if cfg.SubAgent.DefaultModel != "gpt-5.4-mini" {
		t.Errorf("SubAgent.DefaultModel = %q, want gpt-5.4-mini", cfg.SubAgent.DefaultModel)
	}
	if cfg.SubAgent.DefaultEffort != "" {
		t.Errorf("SubAgent.DefaultEffort = %q, want empty string", cfg.SubAgent.DefaultEffort)
	}
	if cfg.SubAgent.MaxConcurrent != 1 {
		t.Errorf("SubAgent.MaxConcurrent = %d, want 1", cfg.SubAgent.MaxConcurrent)
	}
}

func TestLoadConfig_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("LoadConfig() did not create config file")
	}
}

func TestLoadConfig_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	customConfig := &Config{
		DefaultProvider: "openai",
		DefaultModel:    "gpt-4",
		ProviderModels: map[string]ProviderModelConfig{
			"openai": {DefaultModel: "gpt-4"},
		},
		LoopDetection: LoopDetectionConfig{Threshold: 5},
	}

	data, err := yaml.Marshal(customConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("DefaultProvider = %v, want openai", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("DefaultModel = %v, want gpt-4", cfg.DefaultModel)
	}
	if cfg.LoopDetection.Threshold != 5 {
		t.Errorf("LoopDetection.Threshold = %v, want 5", cfg.LoopDetection.Threshold)
	}
}

func TestLoadConfig_LegacyHooksMigratesToFinalChecks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configYAML := `hooks:
  on_completion:
    - go test ./...
  timeout: 120
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.FinalChecks.Commands) != 1 || cfg.FinalChecks.Commands[0] != "go test ./..." {
		t.Fatalf("FinalChecks.Commands = %v, want [go test ./...]", cfg.FinalChecks.Commands)
	}
	if cfg.FinalChecks.Timeout != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", cfg.FinalChecks.Timeout)
	}
}

func TestLoadConfig_FinalChecksWinsOverLegacyVerificationAndHooks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configYAML := `final_checks:
  commands:
    - make test
  timeout: 300
verification:
  commands:
    - legacy verify
  timeout: 200
hooks:
  on_completion:
    - go test ./...
  timeout: 120
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.FinalChecks.Commands) != 1 || cfg.FinalChecks.Commands[0] != "make test" {
		t.Fatalf("FinalChecks.Commands = %v, want [make test]", cfg.FinalChecks.Commands)
	}
	if cfg.FinalChecks.Timeout != 300 {
		t.Fatalf("FinalChecks.Timeout = %d, want 300", cfg.FinalChecks.Timeout)
	}
}

func TestLoadConfig_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	partialYAML := "default_provider: claude\n"

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(partialYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.DefaultProvider != "claude" {
		t.Errorf("DefaultProvider = %v, want claude", cfg.DefaultProvider)
	}
	if cfg.ProviderModels == nil {
		t.Error("ProviderModels should be populated with defaults")
	}
	if cfg.LoopDetection.Threshold != 3 {
		t.Errorf("LoopDetection.Threshold should default to 3, got %d", cfg.LoopDetection.Threshold)
	}
	if cfg.APIRetry.Count != 3 {
		t.Errorf("APIRetry.Count should default to 3, got %d", cfg.APIRetry.Count)
	}
	if cfg.Compression.Enabled != true {
		t.Errorf("Compression.Enabled should default to true, got %v", cfg.Compression.Enabled)
	}
	if cfg.Compression.TriggerPercent != 80 {
		t.Errorf("Compression.TriggerPercent should default to 80, got %d", cfg.Compression.TriggerPercent)
	}
	if cfg.Compression.KeepRecent != 20 {
		t.Errorf("Compression.KeepRecent should default to 20, got %d", cfg.Compression.KeepRecent)
	}
	if cfg.Compression.TokenThreshold != 0 {
		t.Errorf("Compression.TokenThreshold should default to 0, got %d", cfg.Compression.TokenThreshold)
	}
	if cfg.Compression.Model != "" {
		t.Errorf("Compression.Model should default to empty string, got %q", cfg.Compression.Model)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	invalidYAML := "default_provider: openai\n  invalid: - [\n"

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestSaveConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.DefaultProvider = "gemini"

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gemini") {
		t.Error("SaveConfig() did not save custom provider")
	}
	if !strings.Contains(content, "# XELYON CLI 設定") {
		t.Error("SaveConfig() should include header comment")
	}
}

func TestSaveConfig_Permissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("SaveConfig() file mode = %v, want %v", mode, expectedMode)
	}
}

func TestSaveConfig_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal(".xelyon directory should not exist yet")
	}

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create .xelyon directory")
	}

	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0755)
	if mode != expectedMode {
		t.Errorf("SaveConfig() directory mode = %v, want %v", mode, expectedMode)
	}
}

func TestLSPNilServers_RoundTrip_PreservesSiblingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Enabled = false
	cfg.LSP.SkipInstallPrompt = true
	cfg.LSP.Servers = nil

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if !loaded.LSP.SkipInstallPrompt {
		t.Fatal("LSP.SkipInstallPrompt = false, want true")
	}
	if loaded.LSP.Servers != nil {
		t.Fatalf("LSP.Servers = %#v, want nil", loaded.LSP.Servers)
	}
}

func TestLSPEmptyServers_RoundTrip_PreservesSiblingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Enabled = false
	cfg.LSP.SkipInstallPrompt = true
	cfg.LSP.Servers = map[string]LSPServerConfig{}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if !loaded.LSP.SkipInstallPrompt {
		t.Fatal("LSP.SkipInstallPrompt = false, want true")
	}
	if loaded.LSP.Servers == nil {
		t.Fatal("LSP.Servers = nil, want empty map")
	}
	if got := len(loaded.LSP.Servers); got != 0 {
		t.Fatalf("len(LSP.Servers) = %d, want 0", got)
	}
	if _, ok := loaded.LSP.Servers["go"]; ok {
		t.Fatal("default LSP servers should not be restored for explicit empty map")
	}
}

func TestLSPNonEmptyServers_RoundTrip_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Servers = map[string]LSPServerConfig{
		"go": {
			Command:  "custom-gopls",
			Args:     []string{"serve", "--stdio"},
			Disabled: true,
		},
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := len(loaded.LSP.Servers); got != 1 {
		t.Fatalf("len(LSP.Servers) = %d, want 1", got)
	}

	server, ok := loaded.LSP.Servers["go"]
	if !ok {
		t.Fatal("LSP.Servers[\"go\"] not found")
	}
	if server.Command != "custom-gopls" {
		t.Fatalf("LSP.Servers[\"go\"].Command = %q, want %q", server.Command, "custom-gopls")
	}
	if len(server.Args) != 2 || server.Args[0] != "serve" || server.Args[1] != "--stdio" {
		t.Fatalf("LSP.Servers[\"go\"].Args = %#v, want %#v", server.Args, []string{"serve", "--stdio"})
	}
	if !server.Disabled {
		t.Fatal("LSP.Servers[\"go\"].Disabled = false, want true")
	}
}

func TestValidateModel(t *testing.T) {
	tests := []string{"any-model", "gpt-4", "deepseek-coder", ""}
	for _, model := range tests {
		if !ValidateModel(model) {
			t.Errorf("ValidateModel(%q) should always return true", model)
		}
	}
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		checkFn func(*testing.T, *Config)
	}{
		{
			name: "LoopThreshold",
			envVars: map[string]string{
				"XELYON_LOOP_THRESHOLD": "5",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 5 {
					t.Errorf("LoopDetection.Threshold = %d, want 5", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name: "Invalid LoopThreshold",
			envVars: map[string]string{
				"XELYON_LOOP_THRESHOLD": "invalid",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 3 {
					t.Errorf("LoopDetection.Threshold should remain default (3), got %d", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name: "Multiple env vars",
			envVars: map[string]string{
				"XELYON_DIFF_CONTEXT_LINES": "0",
			},
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Diff.ContextLines != 0 {
					t.Errorf("Diff.ContextLines = %d, want 0", cfg.Diff.ContextLines)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg := DefaultConfig()
			cfg.ApplyEnvironmentOverrides()

			tt.checkFn(t, cfg)
		})
	}
}

func TestApplyEnvironmentOverrides_InvalidValues_Warn(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"non-numeric loop threshold", "XELYON_LOOP_THRESHOLD", "abc"},
		{"negative loop threshold", "XELYON_LOOP_THRESHOLD", "-1"},
		{"zero loop threshold", "XELYON_LOOP_THRESHOLD", "0"},
		{"non-numeric diff lines", "XELYON_DIFF_CONTEXT_LINES", "many"},
		{"negative diff lines", "XELYON_DIFF_CONTEXT_LINES", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			cfg := DefaultConfig()
			cfg.ApplyEnvironmentOverrides()

			w.Close()
			os.Stderr = oldStderr

			var buf [1024]byte
			n, _ := r.Read(buf[:])
			r.Close()
			output := string(buf[:n])

			if !strings.Contains(output, "Warning") || !strings.Contains(output, tt.envKey) {
				t.Errorf("Expected warning for %s=%q on stderr, got: %q", tt.envKey, tt.envVal, output)
			}
		})
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	tests := []struct {
		name          string
		loopThreshold *int
		diffLines     *int
		checkFn       func(*testing.T, *Config)
	}{
		{
			name:          "nil pointers",
			loopThreshold: nil,
			diffLines:     nil,
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 3 {
					t.Errorf("LoopDetection.Threshold should remain 3, got %d", cfg.LoopDetection.Threshold)
				}
			},
		},
		{
			name:          "valid values",
			loopThreshold: func() *int { v := 5; return &v }(),
			diffLines:     func() *int { v := 20; return &v }(),
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.LoopDetection.Threshold != 5 {
					t.Errorf("LoopDetection.Threshold = %d, want 5", cfg.LoopDetection.Threshold)
				}
				if cfg.Diff.ContextLines != 20 {
					t.Errorf("Diff.ContextLines = %d, want 20", cfg.Diff.ContextLines)
				}
			},
		},
		{
			name:      "diffLines = 0 is valid",
			diffLines: func() *int { v := 0; return &v }(),
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Diff.ContextLines != 0 {
					t.Errorf("Diff.ContextLines should accept 0, got %d", cfg.Diff.ContextLines)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ApplyFlagOverrides(tt.loopThreshold, tt.diffLines)

			tt.checkFn(t, cfg)
		})
	}
}

func TestIsResponsesAPIModel(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		model    string
		expected bool
	}{
		{"gpt-5.2-codex", true},
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

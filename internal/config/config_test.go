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

	// デフォルト値を確認
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "deepseek-chat" {
		t.Errorf("DefaultModel = %v, want deepseek-chat", cfg.DefaultModel)
	}

	// Provider models
	if cfg.ProviderModels == nil {
		t.Fatal("ProviderModels is nil")
	}

	expectedProviders := []string{"deepseek", "openai", "gemini", "claude", "ollama", "groq", "openrouter", "bedrock"}
	for _, provider := range expectedProviders {
		if _, ok := cfg.ProviderModels[provider]; !ok {
			t.Errorf("ProviderModels missing provider: %s", provider)
		}
	}

	// Compression
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

	// LoopDetection
	if cfg.LoopDetection.Threshold != 3 {
		t.Errorf("LoopDetection.Threshold = %d, want 3", cfg.LoopDetection.Threshold)
	}

	// APIRetry
	if cfg.APIRetry.Count != 3 {
		t.Errorf("APIRetry.Count = %d, want 3", cfg.APIRetry.Count)
	}
	if cfg.APIRetry.InitialDelay != 1 {
		t.Errorf("APIRetry.InitialDelay = %d, want 1", cfg.APIRetry.InitialDelay)
	}
	if cfg.APIRetry.MaxDelay != 30 {
		t.Errorf("APIRetry.MaxDelay = %d, want 30", cfg.APIRetry.MaxDelay)
	}

	// Diff
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
	// テスト用のHOMEディレクトリを設定
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}

	// デフォルト設定が返されることを確認
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %v, want deepseek", cfg.DefaultProvider)
	}

	// 設定ファイルが作成されたことを確認
	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("LoadConfig() did not create config file")
	}
}

func TestLoadConfig_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// カスタム設定ファイルを作成
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

	// 読み込み
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// カスタム値を確認
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

func TestLoadConfig_Partial(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// 部分的な設定ファイル（プロバイダーのみ）
	partialYAML := "default_provider: claude\n"

	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(partialYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// 指定された値
	if cfg.DefaultProvider != "claude" {
		t.Errorf("DefaultProvider = %v, want claude", cfg.DefaultProvider)
	}

	// デフォルト値が適用されることを確認
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

	// 不正なYAML
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

	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// ファイルが作成されたことを確認
	configPath := filepath.Join(tmpDir, ".xelyon", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file")
	}

	// 内容を確認
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gemini") {
		t.Error("SaveConfig() did not save custom provider")
	}

	// ヘッダーコメントを確認
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

	// パーミッション 0600
	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("SaveConfig() file mode = %v, want %v", mode, expectedMode)
	}
}

func TestSaveConfig_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// .xelyonディレクトリが存在しないことを確認
	configDir := filepath.Join(tmpDir, ".xelyon")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal(".xelyon directory should not exist yet")
	}

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// ディレクトリが作成されたことを確認
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create .xelyon directory")
	}

	// ディレクトリパーミッション 0755
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

func TestGetModelForProvider(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "deepseek", provider: "deepseek", want: "deepseek-chat"},
		{name: "openai", provider: "openai", want: "gpt-5.4"},
		{name: "claude", provider: "claude", want: "claude-sonnet-4-6"},
		{name: "ollama", provider: "ollama", want: "qwen2.5-coder:7b"},
		{name: "groq", provider: "groq", want: "meta-llama/llama-4-scout-17b-16e-instruct"},
		{name: "unknown", provider: "unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetModelForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("GetModelForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestValidateModelForProvider(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		provider string
		model    string
		want     bool
	}{
		{name: "valid provider", provider: "deepseek", model: "any-model", want: true},
		{name: "invalid provider", provider: "unknown", model: "any-model", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ValidateModelForProvider(tt.provider, tt.model)
			if got != tt.want {
				t.Errorf("ValidateModelForProvider(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
			}
		})
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
			// 環境変数を設定
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

			// Capture stderr
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
		// GPT-5 シリーズ（prefix マッチ）
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
		// GPT-4o シリーズ（prefix マッチ）
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"gpt-4o-audio-preview", true},
		// o-series reasoning モデル（prefix マッチ）
		{"o1", true},
		{"o1-mini", true},
		{"o1-pro", true},
		{"o3", true},
		{"o3-mini", true},
		{"o3-pro", true},
		{"o4-mini", true},
		// Chat Completions API を使用するモデル（Responses API 非対応）
		{"gpt-4-turbo", false},
		{"gpt-4", false},
		{"gpt-3.5-turbo", false},
		// prefix 誤マッチ防止
		{"openai-custom", false},
		// 存在しないモデル
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

	// カスタムモデルを追加
	cfg.OpenAI.ResponsesAPIModels = append(cfg.OpenAI.ResponsesAPIModels, "custom-codex-model")

	// カスタムモデルが認識されることを確認
	if !cfg.IsResponsesAPIModel("custom-codex-model") {
		t.Error("IsResponsesAPIModel() should return true for custom model")
	}
}

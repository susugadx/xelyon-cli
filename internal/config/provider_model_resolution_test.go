package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
		{name: "anthropic alias", provider: "anthropic", want: "claude-sonnet-4-6"},
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

func TestFindProviderBySelectedModel_PrefersDefaultProviderConfigSpelling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.DefaultModel = "claude-custom"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEmpty, nil)
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("claude-custom"); got != "anthropic" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "claude-custom", got, "anthropic")
	}
}

func TestFindProviderBySelectedModel_FindsAnthropicAliasWhenDisplayProvidersExcludeIt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("anthropic-custom"); got != "anthropic" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "anthropic-custom", got, "anthropic")
	}
}

func TestFindProviderBySelectedModel_ContinuesScanningAliasSiblingAfterDefaultAliasMiss(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "corp-c"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.FindProviderBySelectedModel("corp-c"); got != "claude" {
		t.Fatalf("FindProviderBySelectedModel(%q) = %q, want %q", "corp-c", got, "claude")
	}
	if got := cfg.RuntimeProviderConfigKey("claude", "corp-c"); got != "claude" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "claude", "corp-c", got, "claude")
	}
}

func TestRuntimeProviderConfigKey_PrefersRequestedAliasWhenBothAliasEntriesSelectSameModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {
			DefaultModel:     "shared-custom",
			AnthropicVersion: "2099-01-01",
		},
		"claude": {
			DefaultModel:     "shared-custom",
			AnthropicVersion: "2024-01-01",
		},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.RuntimeProviderConfigKey("anthropic", "shared-custom"); got != "anthropic" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "anthropic", "shared-custom", got, "anthropic")
	}
	if got := cfg.RuntimeProviderConfigKey("claude", "shared-custom"); got != "claude" {
		t.Fatalf("RuntimeProviderConfigKey(%q, %q) = %q, want %q", "claude", "shared-custom", got, "claude")
	}
}

func TestResolveProviderForModel_PreservesConfiguredOwnerBeforeNameInference(t *testing.T) {
	tests := []struct {
		name            string
		defaultProvider string
		defaultModel    string
		state           providerModelSectionState
		raw             map[string]ProviderModelConfig
		currentProvider string
		model           string
		want            string
	}{
		{
			name:            "anthropic alias resolves to claude runtime",
			defaultProvider: "anthropic",
			defaultModel:    "claude-custom",
			state:           providerModelSectionStateExplicitEmpty,
			currentProvider: "openai",
			model:           "claude-custom",
			want:            "claude",
		},
		{
			name:            "ollama keeps opaque local model",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateAbsent,
			currentProvider: "ollama",
			model:           "deepseek-r1:8b",
			want:            "ollama",
		},
		{
			name:            "groq keeps slash-form custom model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			currentProvider: "groq",
			model:           "moonshotai/kimi-k2-instruct",
			want:            "groq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DefaultProvider = tt.defaultProvider
			cfg.DefaultModel = tt.defaultModel
			cfg.providerModelsStore = normalizeProviderModelStore(tt.state, tt.raw)
			cfg.refreshEffectiveProviderModels()

			if got := cfg.ResolveProviderForModel(tt.currentProvider, tt.model); got != tt.want {
				t.Fatalf("ResolveProviderForModel(%q, %q) = %q, want %q", tt.currentProvider, tt.model, got, tt.want)
			}
		})
	}
}

func TestGetModelForProvider_LoadedConfigUsesProviderDefaultWhenProviderOverrideMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: global-custom-model
provider_models:
  openai:
    default_model: gpt-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := cfg.GetModelForProvider("openai"); got != "gpt-custom" {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, "gpt-custom")
	}
	want := DefaultConfig().ProviderModels["deepseek"].DefaultModel
	if got := cfg.GetModelForProvider("deepseek"); got != want {
		t.Fatalf("GetModelForProvider(deepseek) = %q, want provider default %q", got, want)
	}
}

func TestGetModelForProvider_LoadedConfigWithoutProviderModelsUsesProviderDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: deepseek-chat
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetModelForProvider("openai"); got != want {
		t.Fatalf("GetModelForProvider(openai) = %q, want %q", got, want)
	}
	if got := cfg.ResolveModelForProvider("openai"); got != want {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want %q", got, want)
	}
}

func TestGetModelForProvider_LoadedConfigWithNonModelOverrideUsesMergedDefaultModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: openai
default_model: deepseek-chat
provider_models:
  openai:
    max_output_tokens: 99999
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.GetModelForProvider("openai"); got != want {
		t.Fatalf("GetModelForProvider(openai) = %q, want merged default %q", got, want)
	}
	if got := cfg.GetSelectedModelForProvider("openai"); got != want {
		t.Fatalf("GetSelectedModelForProvider(openai) = %q, want %q", got, want)
	}
}

func TestResolveModelForProvider_LoadedConfigUsesProviderDefaultWhenProviderOverrideMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
default_model: global-custom-model
provider_models:
  deepseek:
    default_model: deepseek-custom
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	want := DefaultConfig().ProviderModels["openai"].DefaultModel
	if got := cfg.ResolveModelForProvider("openai"); got != want {
		t.Fatalf("ResolveModelForProvider(openai) = %q, want provider default %q", got, want)
	}
}

func TestGetProviderModelConfig_AliasOverridesCanonicalDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel:    "anthropic-custom",
		MaxOutputTokens: 12345,
		ModelOverrides: map[string]ModelOverride{
			"anthropic-custom": {MaxOutputTokens: 4321},
		},
	})

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should find anthropic alias config")
	}
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.MaxOutputTokens != 12345 {
		t.Fatalf("MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, 12345)
	}
	if pm.AnthropicVersion != DefaultConfig().ProviderModels["claude"].AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want claude default %q", pm.AnthropicVersion, DefaultConfig().ProviderModels["claude"].AnthropicVersion)
	}
	if got := pm.ModelOverrides["anthropic-custom"].MaxOutputTokens; got != 4321 {
		t.Fatalf("ModelOverrides[anthropic-custom].MaxOutputTokens = %d, want %d", got, 4321)
	}
}

func TestGetProviderModelConfig_PrefersRequestedAliasFieldsWhenBothAliasKeysExist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {
			DefaultModel:     "anthropic-custom",
			AnthropicVersion: "2099-01-01",
			AnthropicBeta:    []string{"alias-beta"},
		},
		"claude": {
			DefaultModel: DefaultConfig().ProviderModels["claude"].DefaultModel,
		},
	})
	cfg.refreshEffectiveProviderModels()

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should succeed")
	}
	if pm.DefaultModel != DefaultConfig().ProviderModels["claude"].DefaultModel {
		t.Fatalf("DefaultModel = %q, want default claude model %q", pm.DefaultModel, DefaultConfig().ProviderModels["claude"].DefaultModel)
	}
	if pm.AnthropicVersion != DefaultConfig().ProviderModels["claude"].AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want default claude version %q", pm.AnthropicVersion, DefaultConfig().ProviderModels["claude"].AnthropicVersion)
	}
	if len(pm.AnthropicBeta) != 0 {
		t.Fatalf("AnthropicBeta = %v, want empty when exact claude key is explicit", pm.AnthropicBeta)
	}

	if pmAnthropic, ok := cfg.GetProviderModelConfig("anthropic"); !ok {
		t.Fatal("GetProviderModelConfig(anthropic) should succeed")
	} else {
		if pmAnthropic.DefaultModel != "anthropic-custom" {
			t.Fatalf("GetProviderModelConfig(anthropic).DefaultModel = %q, want %q", pmAnthropic.DefaultModel, "anthropic-custom")
		}
		if pmAnthropic.AnthropicVersion != "2099-01-01" {
			t.Fatalf("GetProviderModelConfig(anthropic).AnthropicVersion = %q, want %q", pmAnthropic.AnthropicVersion, "2099-01-01")
		}
		if len(pmAnthropic.AnthropicBeta) != 1 || pmAnthropic.AnthropicBeta[0] != "alias-beta" {
			t.Fatalf("GetProviderModelConfig(anthropic).AnthropicBeta = %v, want [alias-beta]", pmAnthropic.AnthropicBeta)
		}
	}
}

func TestApplyDefaults_FillsAnthropicAliasProviderModelFields(t *testing.T) {
	cfg := &Config{
		ProviderModels: map[string]ProviderModelConfig{
			"anthropic": {DefaultModel: "anthropic-custom"},
		},
	}
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateInMemoryEffectiveOnly, nil)

	applyDefaults(cfg)

	pm := cfg.ProviderModels["anthropic"]
	claudeDefaults := DefaultConfig().ProviderModels["claude"]
	if pm.DefaultModel != "anthropic-custom" {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, "anthropic-custom")
	}
	if pm.MaxOutputTokens != claudeDefaults.MaxOutputTokens {
		t.Fatalf("MaxOutputTokens = %d, want %d", pm.MaxOutputTokens, claudeDefaults.MaxOutputTokens)
	}
	if pm.AnthropicVersion != claudeDefaults.AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want %q", pm.AnthropicVersion, claudeDefaults.AnthropicVersion)
	}
}

func TestLoadConfig_ExplicitCanonicalProviderModelShadowsAlias(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: claude
provider_models:
  anthropic:
    default_model: anthropic-custom
    anthropic_version: 2099-01-01
    anthropic_beta:
      - alias-beta
  claude:
    default_model: claude-sonnet-4-6
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	pm, ok := cfg.GetProviderModelConfig("claude")
	if !ok {
		t.Fatal("GetProviderModelConfig(claude) should succeed")
	}
	defaultClaude := DefaultConfig().ProviderModels["claude"]
	if pm.DefaultModel != defaultClaude.DefaultModel {
		t.Fatalf("DefaultModel = %q, want %q", pm.DefaultModel, defaultClaude.DefaultModel)
	}
	if pm.AnthropicVersion != defaultClaude.AnthropicVersion {
		t.Fatalf("AnthropicVersion = %q, want %q", pm.AnthropicVersion, defaultClaude.AnthropicVersion)
	}
	if len(pm.AnthropicBeta) != 0 {
		t.Fatalf("AnthropicBeta = %v, want empty", pm.AnthropicBeta)
	}

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}
}

func TestGetProviderModelConfig_PrefersDefaultProviderAliasSpellingWhenBothAliasKeysExist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})

	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if key, ok := cfg.ProviderModelWriteKey("claude"); !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	} else if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}
}

func TestGetSelectedModelForProvider_PrefersRequestedAnthropicAliasOverCanonicalWhenDefaultProviderUnrelated(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.providerModelsStore = normalizeProviderModelStore(providerModelSectionStateExplicitEntries, map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude":    {DefaultModel: "claude-custom"},
	})
	cfg.refreshEffectiveProviderModels()

	if got := cfg.GetSelectedModelForProvider("anthropic"); got != "anthropic-custom" {
		t.Fatalf("GetSelectedModelForProvider(anthropic) = %q, want %q", got, "anthropic-custom")
	}
	if got := cfg.GetSelectedModelForProvider("claude"); got != "claude-custom" {
		t.Fatalf("GetSelectedModelForProvider(claude) = %q, want %q", got, "claude-custom")
	}
}

func TestResolveModelForProvider_AnthropicAliasMatchesClaudeDefaultProvider(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: claude
default_model: claude-custom
provider_models: {}
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := cfg.ResolveModelForProvider("anthropic"); got != "claude-custom" {
		t.Fatalf("ResolveModelForProvider(anthropic) = %q, want %q", got, "claude-custom")
	}
}

func TestGetSelectedModelForProvider_Matrix(t *testing.T) {
	openAIDefault := DefaultConfig().ProviderModels["openai"].DefaultModel
	claudeDefault := DefaultConfig().ProviderModels["claude"].DefaultModel
	ollamaDefault := DefaultConfig().ProviderModels["ollama"].DefaultModel

	tests := []struct {
		name            string
		defaultProvider string
		defaultModel    string
		state           providerModelSectionState
		raw             map[string]ProviderModelConfig
		provider        string
		want            string
	}{
		{
			name:            "absent uses global default for default provider",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            "gpt-global",
		},
		{
			name:            "absent ignores known model from different provider",
			defaultProvider: "openai",
			defaultModel:    "deepseek-chat",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            openAIDefault,
		},
		{
			name:            "absent uses provider default for non default provider",
			defaultProvider: "deepseek",
			defaultModel:    "deepseek-global",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            openAIDefault,
		},
		{
			name:            "absent keeps ollama local model even if it looks foreign",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateAbsent,
			provider:        "ollama",
			want:            "deepseek-r1:8b",
		},
		{
			name:            "explicit empty keeps ollama arbitrary model names",
			defaultProvider: "ollama",
			defaultModel:    "gpt-oss:20b",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "ollama",
			want:            "gpt-oss:20b",
		},
		{
			name:            "explicit empty keeps alias default provider fallback",
			defaultProvider: "claude",
			defaultModel:    "claude-global",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "anthropic",
			want:            "claude-global",
		},
		{
			name:            "explicit empty does not leak global model to other provider",
			defaultProvider: "deepseek",
			defaultModel:    "deepseek-global",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "anthropic",
			want:            claudeDefault,
		},
		{
			name:            "explicit entry shadows top level default",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "openai",
			want:     "gpt-explicit",
		},
		{
			name:            "partial explicit entries keep default provider global model",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "openai",
			want:     "gpt-global",
		},
		{
			name:            "partial explicit entries ignore known model from different provider for default provider",
			defaultProvider: "openai",
			defaultModel:    "deepseek-chat",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {MaxOutputTokens: 999},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "partial explicit entries ignore custom model already owned by different provider",
			defaultProvider: "openai",
			defaultModel:    "custom-shared",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"claude": {DefaultModel: "custom-shared"},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "partial explicit entries keep provider default for unrelated provider",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "claude",
			want:     claudeDefault,
		},
		{
			name:            "partial explicit entries still keep ollama custom model",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "ollama",
			want:     "deepseek-r1:8b",
		},
		{
			name:            "partial explicit entries keep ollama provider default for unrelated provider",
			defaultProvider: "ollama",
			defaultModel:    "gpt-oss:20b",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "explicit empty still keeps ollama built-in default for unrelated provider",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "claude",
			want:            claudeDefault,
		},
		{
			name:            "absent still exposes built-in ollama default when no global override",
			defaultProvider: "ollama",
			defaultModel:    "",
			state:           providerModelSectionStateAbsent,
			provider:        "ollama",
			want:            ollamaDefault,
		},
		{
			name:            "absent keeps groq slash model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateAbsent,
			provider:        "groq",
			want:            "moonshotai/kimi-k2-instruct",
		},
		{
			name:            "partial explicit entries still keep groq slash model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "groq",
			want:     "moonshotai/kimi-k2-instruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DefaultProvider = tt.defaultProvider
			cfg.DefaultModel = tt.defaultModel
			cfg.providerModelsStore = normalizeProviderModelStore(tt.state, tt.raw)
			cfg.refreshEffectiveProviderModels()

			if got := cfg.GetSelectedModelForProvider(tt.provider); got != tt.want {
				t.Fatalf("GetSelectedModelForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
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
		{name: "anthropic alias", provider: "anthropic", model: "any-model", want: true},
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

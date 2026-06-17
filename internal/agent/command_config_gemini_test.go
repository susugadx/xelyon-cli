package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleConfigCommand_GeminiValidation(t *testing.T) {
	t.Run("unsupported model is rejected before save", func(t *testing.T) {
		cfg := newGeminiConfigCommandTestConfig()
		_, out, saved := runGeminiConfigModelCommand(t, cfg, "gemini-2.0-flash-lite")

		assertGeminiConfigModelRejected(t, cfg, out.String(), saved)
	})

	t.Run("catalog model is validated against post-sync config", func(t *testing.T) {
		cfg := newGeminiConfigCommandTestConfig()
		cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
			"gemini": {CatalogModel: "gemini-2.0-flash-lite"},
		})
		_, out, saved := runGeminiConfigModelCommand(t, cfg, "corp-gemini")

		assertGeminiConfigModelRejected(t, cfg, out.String(), saved)
		if !strings.Contains(out.String(), "catalog_model=gemini-2.0-flash-lite") {
			t.Fatalf("output = %q, want Gemini catalog_model validation error", out.String())
		}
	})

	t.Run("default provider is validated even from another current provider", func(t *testing.T) {
		cfg := newGeminiConfigCommandTestConfig()
		_, out, saved := runGeminiConfigModelCommand(t, cfg, "gemini-2.0-flash-lite", func(agent *Agent) {
			agent.ProviderName = "openai"
			agent.ProviderConfigKey = "openai"
			agent.CurrentModel = "gpt-5.4"
			agent.CurrentProvider = &mockProvider{name: "openai"}
		})

		assertGeminiConfigModelRejected(t, cfg, out.String(), saved)
	})
}

func TestRunInteractiveConfig_GeminiDefaultModelValidation(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newGeminiConfigCommandTestConfig()
	categories := []config.ConfigCategory{{Name: "provider"}}
	field := &config.ConfigField{Path: "default_model", FieldType: config.FieldTypeString}
	menu := &scriptedConfigMenu{
		runResults: []configMenuRunResult{
			{category: &categories[0]},
			{category: nil, err: errors.New("cancel")},
		},
		showResults: []configMenuShowResult{
			{field: field},
			{field: nil, err: errors.New("back")},
		},
		editResults: []configMenuEditResult{
			{value: "gemini-2.0-flash-lite", changed: true},
		},
	}

	buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
		return categories
	}
	newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		return menu
	}
	var setCalls int
	setFieldValueForCommand = func(cfg *config.Config, path string, value interface{}) error {
		setCalls++
		cfg.DefaultModel = value.(string)
		return nil
	}
	var saved int
	saveConfigForCommand = func(cfg *config.Config) error {
		saved++
		return nil
	}

	var out bytes.Buffer
	agent := newGeminiConfigCommandTestAgent(cfg, &out)
	runInteractiveConfig(agent, cfg)

	if setCalls != 0 {
		t.Fatalf("setFieldValue calls = %d, want 0", setCalls)
	}
	assertGeminiConfigModelRejected(t, cfg, out.String(), saved)
}

func TestRunInteractiveConfig_GeminiProviderModelsValidation(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newGeminiConfigCommandTestConfig()
	categories := []config.ConfigCategory{{Name: "provider"}}
	field := &config.ConfigField{Path: "provider_models", FieldType: config.FieldTypeStructMap}
	menu := &scriptedConfigMenu{
		runResults: []configMenuRunResult{
			{category: &categories[0]},
			{category: nil, err: errors.New("cancel")},
		},
		showResults: []configMenuShowResult{
			{field: field},
			{field: nil, err: errors.New("back")},
		},
		editResults: []configMenuEditResult{
			{
				changed: true,
				mutate: func() {
					cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
						DefaultModel: "gemini-2.0-flash-lite",
					})
				},
			},
		},
	}

	buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
		return categories
	}
	newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		return menu
	}
	var saved int
	saveConfigForCommand = func(cfg *config.Config) error {
		saved++
		return nil
	}

	var out bytes.Buffer
	agent := newGeminiConfigCommandTestAgent(cfg, &out)
	runInteractiveConfig(agent, cfg)

	if saved != 0 {
		t.Fatalf("save count = %d, want 0", saved)
	}
	if got := cfg.GetExplicitProviderDefaultModel("gemini"); got != "gemini-3.5-flash" {
		t.Fatalf("provider_models.gemini.default_model = %q, want restored gemini-3.5-flash", got)
	}
	if !strings.Contains(out.String(), "provider_models.gemini.default_model") ||
		!strings.Contains(out.String(), "native function calling runtime") {
		t.Fatalf("output = %q, want Gemini provider_models validation error", out.String())
	}
}

func TestRunInteractiveConfig_GeminiDefaultProviderValidation(t *testing.T) {
	withConfigCommandHooks(t)

	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gemini-2.0-flash-lite"
	cfg.ResetProviderModelsForEdit()
	categories := []config.ConfigCategory{{Name: "provider"}}
	field := &config.ConfigField{Path: "default_provider", FieldType: config.FieldTypeString}
	menu := &scriptedConfigMenu{
		runResults: []configMenuRunResult{
			{category: &categories[0]},
			{category: nil, err: errors.New("cancel")},
		},
		showResults: []configMenuShowResult{
			{field: field},
			{field: nil, err: errors.New("back")},
		},
		editResults: []configMenuEditResult{
			{value: "gemini", changed: true},
		},
	}

	buildConfigRegistryForCommand = func(cfg *config.Config) []config.ConfigCategory {
		return categories
	}
	newConfigMenuForCommand = func(cfg *config.Config, categories []config.ConfigCategory, runtime *ui.Runtime) configCommandMenu {
		return menu
	}
	var setCalls int
	setFieldValueForCommand = func(cfg *config.Config, path string, value interface{}) error {
		setCalls++
		cfg.DefaultProvider = value.(string)
		return nil
	}
	var saved int
	saveConfigForCommand = func(cfg *config.Config) error {
		saved++
		return nil
	}

	var out bytes.Buffer
	agent := newConfigCommandTestAgent(cfg, &out)
	cfg.DefaultProvider = "openai"
	runInteractiveConfig(agent, cfg)

	if setCalls != 0 {
		t.Fatalf("setFieldValue calls = %d, want 0", setCalls)
	}
	if saved != 0 {
		t.Fatalf("save count = %d, want 0", saved)
	}
	if cfg.DefaultProvider != "openai" {
		t.Fatalf("DefaultProvider = %q, want unchanged openai", cfg.DefaultProvider)
	}
	if !strings.Contains(out.String(), "default_model") ||
		!strings.Contains(out.String(), "native function calling runtime") {
		t.Fatalf("output = %q, want Gemini default_provider validation error", out.String())
	}
}

func newGeminiConfigCommandTestConfig() *config.Config {
	cfg := newProjectMapDisabledConfig()
	cfg.DefaultProvider = "gemini"
	cfg.DefaultModel = "gemini-3.5-flash"
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{DefaultModel: "gemini-3.5-flash"})
	return cfg
}

func newGeminiConfigCommandTestAgent(cfg *config.Config, out *bytes.Buffer) *Agent {
	agent := newConfigCommandTestAgent(cfg, out)
	cfg.DefaultProvider = "gemini"
	agent.ProviderName = "gemini"
	agent.ProviderConfigKey = "gemini"
	agent.CurrentModel = "gemini-3.5-flash"
	agent.CurrentProvider = &mockProvider{name: "gemini"}
	return agent
}

func runGeminiConfigModelCommand(t *testing.T, cfg *config.Config, model string, mutateAgent ...func(*Agent)) (*Agent, *bytes.Buffer, int) {
	t.Helper()
	withConfigCommandHooks(t)
	loadConfigForCommand = func() (*config.Config, error) {
		return cfg, nil
	}
	var saved int
	saveConfigForCommand = func(cfg *config.Config) error {
		saved++
		return nil
	}

	var out bytes.Buffer
	agent := newGeminiConfigCommandTestAgent(cfg, &out)
	for _, mutate := range mutateAgent {
		mutate(agent)
	}

	if !handleConfigCommand(agent, []string{"model", model}) {
		t.Fatal("handleConfigCommand() = false, want true")
	}
	return agent, &out, saved
}

func assertGeminiConfigModelRejected(t *testing.T, cfg *config.Config, output string, saved int) {
	t.Helper()
	if saved != 0 {
		t.Fatalf("save count = %d, want 0", saved)
	}
	if cfg.DefaultModel != "gemini-3.5-flash" {
		t.Fatalf("DefaultModel = %q, want unchanged gemini-3.5-flash", cfg.DefaultModel)
	}
	if !strings.Contains(output, "gemini function calling is not supported") {
		t.Fatalf("output = %q, want Gemini validation error", output)
	}
}

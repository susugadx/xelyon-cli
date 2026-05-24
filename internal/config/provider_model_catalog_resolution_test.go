package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestModelCatalogName_UsesProviderDefaultCatalogModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("openai", ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]ModelOverride{
			"other-deployment": {CatalogModel: "gpt-5.4-mini"},
		},
	})

	if got := cfg.ModelCatalogName("openai", "corp-gpt-deployment"); got != "gpt-5.4" {
		t.Fatalf("ModelCatalogName(default deployment) = %q, want gpt-5.4", got)
	}
	if got := cfg.ModelCatalogName("openai", "other-deployment"); got != "gpt-5.4-mini" {
		t.Fatalf("ModelCatalogName(model override) = %q, want gpt-5.4-mini", got)
	}
	if got := cfg.ModelCatalogName("openai", "gpt-4.1"); got != "gpt-4.1" {
		t.Fatalf("ModelCatalogName(plain model) = %q, want gpt-4.1", got)
	}
}

func TestModelCatalogName_CanonicalizesGeminiResourceCatalogModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("gemini", ProviderModelConfig{
		DefaultModel: "corp-gemini-flash",
		CatalogModel: "models/Gemini-3.5-Flash",
		ModelOverrides: map[string]ModelOverride{
			"corp-gemini-lite": {CatalogModel: "models/gemini-3.1-flash-lite"},
		},
	})

	if got := cfg.ModelCatalogName("gemini", "corp-gemini-flash"); got != "gemini-3.5-flash" {
		t.Fatalf("ModelCatalogName(gemini default alias) = %q, want gemini-3.5-flash", got)
	}
	if got := cfg.ModelCatalogName("gemini", "corp-gemini-lite"); got != "gemini-3.1-flash-lite" {
		t.Fatalf("ModelCatalogName(gemini override alias) = %q, want gemini-3.1-flash-lite", got)
	}
	if got := cfg.ModelCatalogName("gemini", "models/gemini-3.5-flash"); got != "gemini-3.5-flash" {
		t.Fatalf("ModelCatalogName(gemini resource model) = %q, want gemini-3.5-flash", got)
	}
}

func TestResolveModelCatalog_ProviderCatalogSurvivesDefaultModelOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("openai", ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]ModelOverride{
			"corp-gpt-deployment": {MaxOutputTokens: 8192},
			"other-deployment":    {MaxOutputTokens: 4096},
		},
	})

	defaultOverride := cfg.ResolveModelCatalog("openai", "corp-gpt-deployment")
	if defaultOverride.Model != "gpt-5.4" || defaultOverride.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(default override) = %#v, want provider catalog model", defaultOverride)
	}

	otherOverride := cfg.ResolveModelCatalog("openai", "other-deployment")
	if otherOverride.Model != "other-deployment" || !otherOverride.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(non-default override) = %#v, want configured without catalog", otherOverride)
	}
}

func TestResolveModelCatalog_DefaultModelOverrideCatalogWinsOverProviderCatalog(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("openai", ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]ModelOverride{
			"corp-gpt-deployment": {CatalogModel: "gpt-5.4-mini", MaxOutputTokens: 8192},
		},
	})

	got := cfg.ResolveModelCatalog("openai", "corp-gpt-deployment")
	if got.Model != "gpt-5.4-mini" || got.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(default override catalog) = %#v, want override catalog model", got)
	}
}

func TestResolveModelCatalog_ProviderCatalogSurvivesAliasDefaultModelOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{
		DefaultModel: "corp-claude-deployment",
		CatalogModel: "claude-sonnet-4-6",
		ModelOverrides: map[string]ModelOverride{
			"corp-claude-deployment": {MaxOutputTokens: 8192},
		},
	})

	got := cfg.ResolveModelCatalog("claude", "corp-claude-deployment")
	if got.Model != "claude-sonnet-4-6" || got.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(alias default override) = %#v, want provider catalog model", got)
	}
}

func TestResolveModelCatalog_FlagsConfiguredModelWithoutCatalog(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("openai", ProviderModelConfig{
		DefaultModel: "corp-gpt-5-prod",
		ModelOverrides: map[string]ModelOverride{
			"other-deployment": {MaxOutputTokens: 1234},
			"mini-deployment":  {CatalogModel: "gpt-5.4-mini"},
		},
	})

	defaultAlias := cfg.ResolveModelCatalog("openai", "corp-gpt-5-prod")
	if defaultAlias.Model != "corp-gpt-5-prod" || !defaultAlias.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(default alias) = %#v, want configured without catalog", defaultAlias)
	}

	overrideAlias := cfg.ResolveModelCatalog("openai", "other-deployment")
	if overrideAlias.Model != "other-deployment" || !overrideAlias.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(override alias) = %#v, want configured without catalog", overrideAlias)
	}

	knownOverride := cfg.ResolveModelCatalog("openai", "mini-deployment")
	if knownOverride.Model != "gpt-5.4-mini" || knownOverride.ConfiguredWithoutCatalog {
		t.Fatalf("ResolveModelCatalog(override catalog) = %#v, want catalog model", knownOverride)
	}
}

func TestIsProviderResponsesAPIModel_UsesCatalogModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelConfig("openai", ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]ModelOverride{
			"mini-deployment": {CatalogModel: "gpt-4o-mini"},
		},
	})

	if !cfg.IsProviderResponsesAPIModel("openai", "corp-gpt-deployment") {
		t.Fatal("IsProviderResponsesAPIModel(default deployment) = false, want true via catalog_model")
	}
	if !cfg.IsProviderResponsesAPIModel("openai", "mini-deployment") {
		t.Fatal("IsProviderResponsesAPIModel(model override deployment) = false, want true via catalog_model")
	}
	if cfg.IsProviderResponsesAPIModel("groq", "gpt-5.4") {
		t.Fatal("IsProviderResponsesAPIModel(groq, gpt-5.4) = true, want false for provider without Responses API")
	}
}

func TestProviderModelCatalogFields_YAMLRoundTrip(t *testing.T) {
	input := []byte(`
provider_models:
  openai:
    default_model: corp-gpt-deployment
    catalog_model: gpt-5.4
    model_overrides:
      other-deployment:
        catalog_model: gpt-5.4-mini
        max_output_tokens: 1234
`)
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(input, cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	applyDefaults(cfg)

	pm := cfg.ProviderModels["openai"]
	if pm.CatalogModel != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].CatalogModel = %q, want gpt-5.4", pm.CatalogModel)
	}
	override := pm.ModelOverrides["other-deployment"]
	if override.CatalogModel != "gpt-5.4-mini" || override.MaxOutputTokens != 1234 {
		t.Fatalf("ModelOverrides[other-deployment] = %#v, want catalog_model and max_output_tokens", override)
	}
}

func TestProviderModelsDisplayNameKeyCanonicalizesToOwner(t *testing.T) {
	cfg, err := loadConfigFromData([]byte(`
provider_models:
  Azure OpenAI:
    default_model: corp-gpt55-deployment
    catalog_model: gpt-5.5
    max_output_tokens: 12345
`))
	if err != nil {
		t.Fatalf("loadConfigFromData() error = %v", err)
	}

	if got := cfg.GetSelectedModelForProvider("azure"); got != "corp-gpt55-deployment" {
		t.Fatalf("GetSelectedModelForProvider(azure) = %q, want display-name deployment", got)
	}
	if got := cfg.ModelCatalogName("azure", "corp-gpt55-deployment"); got != "gpt-5.5" {
		t.Fatalf("ModelCatalogName(azure, deployment) = %q, want gpt-5.5", got)
	}
	if _, ok := cfg.ProviderModels["azure openai"]; ok {
		t.Fatalf("ProviderModels contains display-name normalized key: %#v", cfg.ProviderModels)
	}
	if got := cfg.ProviderModels["azure"].MaxOutputTokens; got != 12345 {
		t.Fatalf("ProviderModels[azure].MaxOutputTokens = %d, want 12345", got)
	}

	saved := cfg.ProviderModelsForSave()
	if _, ok := saved["azure"]; !ok {
		t.Fatalf("ProviderModelsForSave() = %#v, want azure key", saved)
	}
	if _, ok := saved["azure openai"]; ok {
		t.Fatalf("ProviderModelsForSave() contains display-name normalized key: %#v", saved)
	}
}

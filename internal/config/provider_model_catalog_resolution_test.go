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

package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type providerValidationTestProvider struct {
	name string
}

func (p *providerValidationTestProvider) Name() string { return p.name }
func (p *providerValidationTestProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}
func (p *providerValidationTestProvider) SupportsImages() bool { return false }
func (p *providerValidationTestProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}
func (p *providerValidationTestProvider) IsFunctionCallingEnabled() bool { return true }

func TestValidateSelectedProviderModel(t *testing.T) {
	origModelFlag := modelFlag
	t.Cleanup(func() {
		modelFlag = origModelFlag
	})
	t.Setenv("XELYON_MODEL", "")

	t.Run("non-azure provider is ignored", func(t *testing.T) {
		err := validateSelectedProviderModel(config.DefaultConfig(), &providerValidationTestProvider{name: "openai"}, "gpt-5.4")
		if err != nil {
			t.Fatalf("validateSelectedProviderModel() error = %v, want nil", err)
		}
	})

	t.Run("azure with explicit provider deployment passes", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
			DefaultModel: "corp-gpt55-deployment",
			CatalogModel: "gpt-5.5",
		})
		if err := validateSelectedProviderModel(cfg, &providerValidationTestProvider{name: "Azure OpenAI"}, "corp-gpt55-deployment"); err != nil {
			t.Fatalf("validateSelectedProviderModel() error = %v, want nil", err)
		}
	})

	t.Run("azure placeholder without explicit deployment fails", func(t *testing.T) {
		modelFlag = ""
		_ = os.Unsetenv("XELYON_MODEL")
		err := validateSelectedProviderModel(config.DefaultConfig(), &providerValidationTestProvider{name: "Azure OpenAI"}, "azure-gpt-5.4")
		if err == nil {
			t.Fatal("validateSelectedProviderModel() error = nil, want Azure deployment guidance")
		}
		if !strings.Contains(err.Error(), "provider_models.azure.default_model") ||
			!strings.Contains(err.Error(), "pass --model <deployment>") {
			t.Fatalf("validateSelectedProviderModel() error = %v, want config and CLI model guidance", err)
		}
	})

	t.Run("azure placeholder with explicit model flag passes", func(t *testing.T) {
		modelFlag = "azure-gpt-5.4"
		if err := validateSelectedProviderModel(config.DefaultConfig(), &providerValidationTestProvider{name: "Azure OpenAI"}, "azure-gpt-5.4"); err != nil {
			t.Fatalf("validateSelectedProviderModel() error = %v, want nil with explicit --model", err)
		}
	})

	t.Run("azure placeholder with explicit saved session model passes", func(t *testing.T) {
		modelFlag = ""
		_ = os.Unsetenv("XELYON_MODEL")
		err := validateSelectedProviderModelWithContext(config.DefaultConfig(), &providerValidationTestProvider{name: "Azure OpenAI"}, "azure-gpt-5.4", selectedProviderModelValidationContext{
			explicitModel:     true,
			providerConfigKey: "azure",
		})
		if err != nil {
			t.Fatalf("validateSelectedProviderModelWithContext() error = %v, want nil with saved session model", err)
		}
	})

	t.Run("gemini unsupported function calling model fails", func(t *testing.T) {
		modelFlag = ""
		err := validateSelectedProviderModel(config.DefaultConfig(), &providerValidationTestProvider{name: "gemini"}, "gemini-2.0-flash-lite")
		if err == nil {
			t.Fatal("validateSelectedProviderModel() error = nil, want Gemini function calling guidance")
		}
	})

	t.Run("gemini unsupported catalog model fails", func(t *testing.T) {
		modelFlag = ""
		cfg := config.DefaultConfig()
		cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
			ModelOverrides: map[string]config.ModelOverride{
				"corp-lite": {CatalogModel: "gemini-2.0-flash-lite"},
			},
		})
		err := validateSelectedProviderModel(cfg, &providerValidationTestProvider{name: "gemini"}, "corp-lite")
		if err == nil {
			t.Fatal("validateSelectedProviderModel() error = nil, want Gemini catalog_model guidance")
		}
	})
}

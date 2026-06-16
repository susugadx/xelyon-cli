package cliruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type validationProvider struct {
	name string
}

func (p validationProvider) Name() string                   { return p.name }
func (p validationProvider) SupportsImages() bool           { return false }
func (p validationProvider) IsFunctionCallingEnabled() bool { return true }
func (p validationProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}
func (p validationProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}

func TestGetModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DefaultProvider = "openai"
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-custom"}

	if got := GetModel(cfg, ModelSelection{ModelFlag: "flag-model"}); got != "flag-model" {
		t.Fatalf("flag model = %q", got)
	}

	t.Setenv("XELYON_MODEL", "env-model")
	if got := GetModel(cfg, ModelSelection{}); got != "env-model" {
		t.Fatalf("env model = %q", got)
	}
}

func TestValidateSelectedProviderModel(t *testing.T) {
	t.Setenv("XELYON_MODEL", "")

	err := ValidateSelectedProviderModel(config.DefaultConfig(), validationProvider{name: "Azure OpenAI"}, "azure-gpt-5.4", ProviderModelValidationContext{})
	if err == nil {
		t.Fatal("ValidateSelectedProviderModel() error = nil, want Azure deployment guidance")
	}
	if !strings.Contains(err.Error(), "pass --model <deployment>") {
		t.Fatalf("ValidateSelectedProviderModel() error = %v, want CLI guidance", err)
	}

	err = ValidateSelectedProviderModel(config.DefaultConfig(), validationProvider{name: "Azure OpenAI"}, "azure-gpt-5.4", ProviderModelValidationContext{ModelFlag: "azure-gpt-5.4"})
	if err != nil {
		t.Fatalf("ValidateSelectedProviderModel() explicit model error = %v", err)
	}
}

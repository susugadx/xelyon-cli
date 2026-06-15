package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const azureDefaultPlaceholderDeployment = "azure-gpt-5.4"

type selectedProviderModelValidationContext struct {
	explicitModel     bool
	providerConfigKey string
}

func validateSelectedProviderModel(cfg *config.Config, provider api.Provider, model string) error {
	return validateSelectedProviderModelWithContext(cfg, provider, model, selectedProviderModelValidationContext{
		explicitModel: hasExplicitCLIModelSelection(),
	})
}

func validateSelectedProviderModelWithContext(cfg *config.Config, provider api.Provider, model string, validation selectedProviderModelValidationContext) error {
	if provider == nil {
		return nil
	}
	runtimeProvider := config.CanonicalProviderName(provider.Name())
	if runtimeProvider == "gemini" {
		providerKey := strings.TrimSpace(validation.providerConfigKey)
		if providerKey == "" {
			providerKey = runtimeProvider
		}
		return config.ValidateGeminiFunctionCallingSelection(cfg, providerKey, model)
	}
	if runtimeProvider != "azure" {
		return nil
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or pass --model <deployment>")
	}
	if !strings.EqualFold(model, azureDefaultPlaceholderDeployment) {
		return nil
	}

	if hasExplicitSelectedModel(validation) {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model or pass --model <deployment>")
	}
	if providerKey := strings.TrimSpace(validation.providerConfigKey); providerKey != "" {
		if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(providerKey)); explicit != "" {
			return nil
		}
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return nil
	}

	return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model or pass --model <deployment>")
}

func hasExplicitSelectedModel(validation selectedProviderModelValidationContext) bool {
	return validation.explicitModel || hasExplicitCLIModelSelection()
}

func hasExplicitCLIModelSelection() bool {
	return strings.TrimSpace(modelFlag) != "" || strings.TrimSpace(os.Getenv("XELYON_MODEL")) != ""
}

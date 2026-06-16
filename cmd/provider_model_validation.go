package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

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

	err := config.ValidateAzureDeploymentSelection(cfg, validation.providerConfigKey, model, hasExplicitSelectedModel(validation))
	return selectedProviderModelValidationError(err)
}

func selectedProviderModelValidationError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSuffix(err.Error(), ".")
	return fmt.Errorf("%s or pass --model <deployment>", message)
}

func hasExplicitSelectedModel(validation selectedProviderModelValidationContext) bool {
	return validation.explicitModel || hasExplicitCLIModelSelection()
}

func hasExplicitCLIModelSelection() bool {
	return strings.TrimSpace(modelFlag) != "" || strings.TrimSpace(os.Getenv("XELYON_MODEL")) != ""
}

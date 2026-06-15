package config

import (
	"fmt"
	"strings"
)

const azureDefaultPlaceholderDeployment = "azure-gpt-5.4"

// ValidateAzureDeploymentSelection は Azure OpenAI の deployment 選択が実行可能か検証する。
func ValidateAzureDeploymentSelection(cfg *Config, providerConfigKey, model string, explicitModel bool) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model")
	}
	if !strings.EqualFold(model, azureDefaultPlaceholderDeployment) {
		return nil
	}
	if explicitModel {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model")
	}
	providerConfigKey = strings.TrimSpace(providerConfigKey)
	if providerConfigKey != "" {
		if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(providerConfigKey)); explicit != "" {
			return nil
		}
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return nil
	}
	return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model")
}

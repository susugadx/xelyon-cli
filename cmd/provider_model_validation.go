package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const azureDefaultPlaceholderDeployment = "azure-gpt-5.4"

func validateSelectedProviderModel(cfg *config.Config, provider api.Provider, model string) error {
	if provider == nil {
		return nil
	}
	if config.CanonicalProviderName(provider.Name()) != "azure" {
		return nil
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or pass --model <deployment>")
	}
	if !strings.EqualFold(model, azureDefaultPlaceholderDeployment) {
		return nil
	}

	if strings.TrimSpace(modelFlag) != "" || strings.TrimSpace(os.Getenv("XELYON_MODEL")) != "" {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model or pass --model <deployment>")
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return nil
	}

	return fmt.Errorf("azure OpenAI deployment is not configured. Set provider_models.azure.default_model or pass --model <deployment>")
}

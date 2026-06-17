package cmd

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/cliruntime"
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
	return cliruntime.ValidateSelectedProviderModel(cfg, provider, model, cliruntime.ProviderModelValidationContext{
		ExplicitModel:     validation.explicitModel,
		ProviderConfigKey: validation.providerConfigKey,
		ModelFlag:         modelFlag,
	})
}

func hasExplicitCLIModelSelection() bool {
	return cliruntime.HasExplicitCLIModelSelection(modelFlag)
}

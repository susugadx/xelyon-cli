package cmd

import (
	"github.com/spf13/cobra"
	"github.com/susugadx/xelyon-cli/internal/api"
)

func resolveProviderForExecutionMode(cmd *cobra.Command, providerName string, mode executionMode, model string) (api.Provider, error) {
	if executionModeIsInteractive(mode) {
		return resolveInteractiveProvider(providerName)
	}
	provider, err := resolveRequiredProvider(providerName)
	if err == nil {
		return provider, nil
	}
	if mode == executionModeHeadless && isProviderSetupError(providerName, err) {
		return nil, &headlessProviderSetupRequiredError{
			provider: providerName,
			model:    model,
			message:  err.Error(),
		}
	}
	if mode == executionModeHeadless {
		return nil, newHeadlessRuntimeSelectionConfigError(providerName, model, err)
	}
	return nil, err
}

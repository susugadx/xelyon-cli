package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// SwitchProvider はプロバイダーを切り替える
func (a *Agent) SwitchProvider(providerName string) error {
	return a.switchProvider(providerName, "")
}

func (a *Agent) switchProvider(providerName, requestedModel string) error {
	outcome, err := a.SwitchProviderModel(providerName, requestedModel)
	if err != nil {
		return err
	}
	printProviderSwitchOutcome(a, outcome)
	return nil
}

func printProviderSwitchOutcome(agent *Agent, outcome ProviderSwitchOutcome) {
	out := agent.output()
	green.Fprintf(out, "✅ Provider: %s → %s\n", outcome.OldProvider, outcome.NewProvider)
	green.Fprintf(out, "✅ Model: %s → %s\n", outcome.OldModel, outcome.NewModel)
	printRuntimeSwitchContextNotice(agent, outcome.ContextNotice)
}

func validateProviderModelSelection(cfg *config.Config, runtimeProviderName, providerConfigKey, model string, explicitModel bool) error {
	runtimeProvider := config.CanonicalProviderName(runtimeProviderName)
	if runtimeProvider == "gemini" {
		providerKey := strings.TrimSpace(providerConfigKey)
		if providerKey == "" {
			providerKey = runtimeProvider
		}
		return config.ValidateGeminiFunctionCallingSelection(cfg, providerKey, model)
	}
	if runtimeProvider != "azure" {
		return nil
	}
	return config.ValidateAzureDeploymentSelection(cfg, providerConfigKey, model, explicitModel)
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	return config.ProviderHasAvailableCredential(provider)
}

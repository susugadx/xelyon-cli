package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

const azureDefaultPlaceholderDeployment = "azure-gpt-5.4"

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

	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or use '/provider azure <deployment>'")
	}
	if !strings.EqualFold(model, azureDefaultPlaceholderDeployment) {
		return nil
	}
	if explicitModel {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("azure OpenAI deployment is required: set provider_models.azure.default_model or use '/provider azure <deployment>'")
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(providerConfigKey)); explicit != "" {
		return nil
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("azure")); explicit != "" {
		return nil
	}
	return fmt.Errorf(
		"azure OpenAI deployment is not configured. Set provider_models.azure.default_model or run '/provider azure <deployment>'",
	)
}

// IsAPIKeyAvailable は指定されたプロバイダーのAPIキーが利用可能かチェック
func IsAPIKeyAvailable(provider string) bool {
	return config.ProviderHasAvailableCredential(provider)
}

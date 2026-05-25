package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func validateConfigModelChange(agent *Agent, cfg *config.Config, model string) error {
	candidate := config.CloneConfig(cfg)
	candidate.DefaultModel = model
	if agent != nil {
		agent.SyncDefaultModelToProvider(candidate)
	}
	for _, selection := range configModelChangeProviderSelections(agent, candidate) {
		if config.CanonicalProviderName(selection.providerName) != "gemini" {
			continue
		}
		if err := validateProviderModelSelection(candidate, selection.providerName, selection.providerConfigKey, model, true); err != nil {
			return err
		}
	}
	return nil
}

type configModelChangeProviderSelection struct {
	providerName      string
	providerConfigKey string
}

func configModelChangeProviderSelections(agent *Agent, cfg *config.Config) []configModelChangeProviderSelection {
	var selections []configModelChangeProviderSelection
	if agent != nil {
		selections = append(selections, configModelChangeProviderSelection{
			providerName:      agent.ProviderName,
			providerConfigKey: agent.currentProviderConfigKey(),
		})
	}
	if cfg != nil {
		providerName := cfg.DefaultProvider
		providerConfigKey := config.ActiveProviderConfigKey(providerName)
		if providerConfigKey == "" {
			providerConfigKey = config.CanonicalProviderName(providerName)
		}
		selections = append(selections, configModelChangeProviderSelection{
			providerName:      providerName,
			providerConfigKey: providerConfigKey,
		})
	}
	return dedupeConfigModelChangeProviderSelections(selections)
}

func dedupeConfigModelChangeProviderSelections(selections []configModelChangeProviderSelection) []configModelChangeProviderSelection {
	if len(selections) < 2 {
		return selections
	}

	deduped := selections[:0]
	seen := map[configModelChangeProviderSelection]bool{}
	for _, selection := range selections {
		key := configModelChangeProviderSelection{
			providerName:      config.CanonicalProviderName(selection.providerName),
			providerConfigKey: config.ActiveProviderConfigKey(selection.providerConfigKey),
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, selection)
	}
	return deduped
}

func validateGeminiFunctionCallingConfigForSave(cfg *config.Config) error {
	return firstGeminiFunctionCallingValidationError(config.ValidateGeminiFunctionCallingConfig(cfg))
}

func validateGeminiFunctionCallingConfigForSaveIfRelevant(cfg *config.Config, providers ...string) error {
	for _, provider := range providers {
		if config.SameProviderRuntimeIdentity(provider, "gemini") {
			return validateGeminiFunctionCallingConfigForSave(cfg)
		}
	}
	return nil
}

func firstGeminiFunctionCallingValidationError(result config.ValidationResult) error {
	if result.Valid {
		return nil
	}
	for _, issue := range result.Issues {
		if issue.Severity == config.ValidationSeverityError {
			if issue.Suggestion != "" {
				return fmt.Errorf("%s: %s; %s", issue.Field, issue.Message, issue.Suggestion)
			}
			return fmt.Errorf("%s: %s", issue.Field, issue.Message)
		}
	}
	return nil
}

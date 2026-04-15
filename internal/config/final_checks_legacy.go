package config

import "gopkg.in/yaml.v3"

type legacyCompletionHooksConfig struct {
	OnCompletion []string `yaml:"on_completion"`
	Timeout      int      `yaml:"timeout"`
}

type finalChecksCompatibilityEnvelope struct {
	FinalChecks  *FinalChecksConfig           `yaml:"final_checks,omitempty"`
	Verification *FinalChecksConfig           `yaml:"verification,omitempty"`
	Hooks        *legacyCompletionHooksConfig `yaml:"hooks,omitempty"`
}

func loadCompatibleFinalChecks(data []byte) (*FinalChecksConfig, error) {
	var raw finalChecksCompatibilityEnvelope
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	switch {
	case raw.FinalChecks != nil:
		return raw.FinalChecks, nil
	case raw.Verification != nil:
		return raw.Verification, nil
	default:
		return legacyHooksToFinalChecks(raw.Hooks), nil
	}
}

func legacyHooksToFinalChecks(hooks *legacyCompletionHooksConfig) *FinalChecksConfig {
	if hooks == nil || len(hooks.OnCompletion) == 0 {
		return nil
	}
	return &FinalChecksConfig{
		Commands: append([]string(nil), hooks.OnCompletion...),
		Timeout:  hooks.Timeout,
	}
}

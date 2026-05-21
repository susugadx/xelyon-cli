package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type providerHistoryReductionModeResolution struct {
	configured ProviderHistoryReductionMode
	effective  ProviderHistoryReductionMode
	specified  bool
}

func syncProviderHistoryReductionModeFromProjectConfig(runtime *AgentRuntime, projectCfg *config.ProjectConfig) error {
	if runtime == nil {
		return nil
	}
	mode, specified, err := config.ResolveProjectProviderHistoryReductionMode(projectCfg, nil)
	if err != nil {
		return err
	}
	runtime.Options.ProviderHistoryReductionMode = providerHistoryReductionModeFromProjectConfigMode(mode)
	runtime.Options.ProviderHistoryReductionModeSet = specified
	return nil
}

func providerHistoryReductionModeFromProjectConfigMode(mode config.ProjectProviderHistoryReductionMode) ProviderHistoryReductionMode {
	switch mode {
	case config.ProjectProviderHistoryReductionModeDryRun:
		return ProviderHistoryReductionDryRun
	case config.ProjectProviderHistoryReductionModeApply:
		return ProviderHistoryReductionApply
	case config.ProjectProviderHistoryReductionModeAuto:
		return ProviderHistoryReductionAuto
	default:
		return ProviderHistoryReductionDisabled
	}
}

func providerHistoryReductionConfiguredModeForRuntime(runtime *AgentRuntime) (ProviderHistoryReductionMode, bool) {
	if runtime == nil {
		return ProviderHistoryReductionDisabled, false
	}
	opts := runtime.Options
	if opts.ProviderHistoryReductionModeSet {
		return opts.ProviderHistoryReductionMode, true
	}
	if opts.ProviderHistoryReductionMode != ProviderHistoryReductionDisabled {
		return opts.ProviderHistoryReductionMode, true
	}
	if opts.EnableProviderHistoryReduction {
		return ProviderHistoryReductionApply, true
	}
	return ProviderHistoryReductionDisabled, false
}

func providerHistoryReductionModeResolutionForRuntime(runtime *AgentRuntime) providerHistoryReductionModeResolution {
	configured, specified := providerHistoryReductionConfiguredModeForRuntime(runtime)
	return providerHistoryReductionModeResolution{
		configured: configured,
		effective:  providerHistoryReductionEffectiveModeForConfiguredMode(configured),
		specified:  specified,
	}
}

func providerHistoryReductionEffectiveModeForConfiguredMode(mode ProviderHistoryReductionMode) ProviderHistoryReductionMode {
	switch mode {
	case ProviderHistoryReductionApply:
		return ProviderHistoryReductionApply
	case ProviderHistoryReductionDryRun, ProviderHistoryReductionAuto:
		return ProviderHistoryReductionDryRun
	default:
		return ProviderHistoryReductionDisabled
	}
}

package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type providerHistoryReductionModeResolution struct {
	configured ProviderHistoryReductionMode
	effective  ProviderHistoryReductionMode
	specified  bool
}

type providerHistoryRuntimeConfigResolution struct {
	mode             ProviderHistoryReductionMode
	modeSet          bool
	rehydrateContext bool
}

func syncProviderHistoryRuntimeConfigFromProjectConfig(runtime *AgentRuntime, projectCfg *config.ProjectConfig) error {
	if runtime == nil {
		return nil
	}
	resolution, err := resolveProviderHistoryRuntimeConfigFromProjectConfig(projectCfg)
	if err != nil {
		return err
	}
	applyProviderHistoryRuntimeConfigToRuntime(runtime, resolution)
	return nil
}

func resolveProviderHistoryRuntimeConfigFromProjectConfig(projectCfg *config.ProjectConfig) (providerHistoryRuntimeConfigResolution, error) {
	mode, specified, err := resolveProviderHistoryReductionModeFromProjectConfig(projectCfg)
	if err != nil {
		return providerHistoryRuntimeConfigResolution{}, err
	}
	rehydrateContext, err := resolveProviderHistoryRehydrateContextFromProjectConfig(projectCfg)
	if err != nil {
		return providerHistoryRuntimeConfigResolution{}, err
	}
	return providerHistoryRuntimeConfigResolution{
		mode:             mode,
		modeSet:          specified,
		rehydrateContext: rehydrateContext,
	}, nil
}

func resolveProviderHistoryReductionModeFromProjectConfig(projectCfg *config.ProjectConfig) (ProviderHistoryReductionMode, bool, error) {
	mode, specified, err := config.ResolveProjectProviderHistoryReductionMode(projectCfg, nil)
	if err != nil {
		return ProviderHistoryReductionDisabled, false, err
	}
	return providerHistoryReductionModeFromProjectConfigMode(mode), specified, nil
}

func resolveProviderHistoryRehydrateContextFromProjectConfig(projectCfg *config.ProjectConfig) (bool, error) {
	return config.ResolveProjectProviderHistoryRehydrateContext(projectCfg, nil)
}

func applyProviderHistoryRuntimeConfigToRuntime(runtime *AgentRuntime, resolution providerHistoryRuntimeConfigResolution) {
	if runtime == nil {
		return
	}
	applyProviderHistoryReductionModeToRuntime(runtime, resolution.mode, resolution.modeSet)
	runtime.Options.EnableProviderHistoryRehydrateContext = resolution.rehydrateContext
}

func applyProviderHistoryReductionModeToRuntime(runtime *AgentRuntime, mode ProviderHistoryReductionMode, specified bool) {
	if runtime == nil {
		return
	}
	runtime.Options.ProviderHistoryReductionMode = mode
	runtime.Options.ProviderHistoryReductionModeSet = specified
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

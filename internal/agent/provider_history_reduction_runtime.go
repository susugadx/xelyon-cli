package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type providerHistoryReductionModeResolution struct {
	configured ProviderHistoryReductionMode
	effective  ProviderHistoryReductionMode
	specified  bool
}

type providerHistoryRuntimeConfigResolution struct {
	mode               ProviderHistoryReductionMode
	modeSet            bool
	rehydrateContext   bool
	rawOutputArtifacts config.ProviderHistoryRawOutputArtifactsConfig
	rawOutputSet       bool
}

func syncProviderHistoryRuntimeConfigFromProjectConfig(runtime *AgentRuntime, projectCfg *config.ProjectConfig) error {
	if runtime == nil {
		return nil
	}
	resolution, err := resolveProviderHistoryRuntimeConfig(runtime.effectiveConfig(), projectCfg)
	if err != nil {
		return err
	}
	applyProviderHistoryRuntimeConfigToRuntime(runtime, resolution)
	return nil
}

func (a *Agent) syncProviderHistoryRuntimeConfigFromCurrentProject() error {
	if a == nil || a.Runtime == nil {
		return nil
	}
	projectCfg := a.projectConfigStore().LoadForCWD(a.invocationCWD())
	return syncProviderHistoryRuntimeConfigFromProjectConfig(a.Runtime, projectCfg)
}

func resolveProviderHistoryRuntimeConfig(globalCfg *config.Config, projectCfg *config.ProjectConfig) (providerHistoryRuntimeConfigResolution, error) {
	mode, specified, err := resolveProviderHistoryReductionMode(globalCfg, projectCfg)
	if err != nil {
		return providerHistoryRuntimeConfigResolution{}, err
	}
	rehydrateContext, err := resolveProviderHistoryRehydrateContext(globalCfg, projectCfg)
	if err != nil {
		return providerHistoryRuntimeConfigResolution{}, err
	}
	rawOutputArtifacts, rawOutputSet, err := config.ResolveProviderHistoryRawOutputArtifactsConfig(globalCfg, projectCfg)
	if err != nil {
		return providerHistoryRuntimeConfigResolution{}, err
	}
	return providerHistoryRuntimeConfigResolution{
		mode:               mode,
		modeSet:            specified,
		rehydrateContext:   rehydrateContext,
		rawOutputArtifacts: rawOutputArtifacts,
		rawOutputSet:       rawOutputSet,
	}, nil
}

func resolveProviderHistoryReductionMode(globalCfg *config.Config, projectCfg *config.ProjectConfig) (ProviderHistoryReductionMode, bool, error) {
	mode, specified, err := config.ResolveProviderHistoryReductionMode(globalCfg, projectCfg, nil)
	if err != nil {
		return ProviderHistoryReductionDisabled, false, err
	}
	return providerHistoryReductionModeFromProjectConfigMode(mode), specified, nil
}

func resolveProviderHistoryRehydrateContext(globalCfg *config.Config, projectCfg *config.ProjectConfig) (bool, error) {
	return config.ResolveProviderHistoryRehydrateContext(globalCfg, projectCfg, nil)
}

func applyProviderHistoryRuntimeConfigToRuntime(runtime *AgentRuntime, resolution providerHistoryRuntimeConfigResolution) {
	if runtime == nil {
		return
	}
	if runtime.Options.ProviderHistoryRawOutputArtifacts != resolution.rawOutputArtifacts {
		runtime.RawOutputArtifactStore = nil
	}
	applyProviderHistoryReductionModeToRuntime(runtime, resolution.mode, resolution.modeSet)
	runtime.Options.EnableProviderHistoryRehydrateContext = resolution.rehydrateContext
	runtime.Options.ProviderHistoryRawOutputArtifacts = resolution.rawOutputArtifacts
	runtime.Options.ProviderHistoryRawOutputArtifactsSet = resolution.rawOutputSet
	runtime.RawOutputArtifactRoot = resolution.rawOutputArtifacts.Root
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

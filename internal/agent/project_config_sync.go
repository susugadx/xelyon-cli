package agent

import "github.com/susugadx/xelyon-cli/internal/config"

type runtimeProjectConfigResolution struct {
	providerHistoryReductionMode    ProviderHistoryReductionMode
	providerHistoryReductionModeSet bool
	finalChecks                     config.FinalChecksConfig
}

// SaveAndSyncProjectConfig は xelyon.yaml を保存し、現在 runtime の project 由来設定を同期する。
func (a *Agent) SaveAndSyncProjectConfig(pc *config.ProjectConfig) error {
	if err := config.SaveProjectConfig(pc); err != nil {
		return err
	}
	a.projectConfigStore().Clear()
	return a.syncRuntimeProjectConfig(pc)
}

func (a *Agent) syncRuntimeProjectConfig(pc *config.ProjectConfig) error {
	if a == nil {
		return nil
	}

	resolution, err := a.resolveRuntimeProjectConfig(pc)
	if err != nil {
		return err
	}
	a.applyRuntimeProjectConfig(resolution)
	a.promptManager().InvalidateProjectMap()
	a.refreshProjectPrompt("")
	return nil
}

func (a *Agent) resolveRuntimeProjectConfig(pc *config.ProjectConfig) (runtimeProjectConfigResolution, error) {
	providerHistoryMode, providerHistoryModeSet, err := resolveProviderHistoryReductionModeFromProjectConfig(pc)
	if err != nil {
		return runtimeProjectConfigResolution{}, err
	}
	finalChecks, err := a.resolveRuntimeProjectFinalChecks(pc)
	if err != nil {
		return runtimeProjectConfigResolution{}, err
	}

	return runtimeProjectConfigResolution{
		providerHistoryReductionMode:    providerHistoryMode,
		providerHistoryReductionModeSet: providerHistoryModeSet,
		finalChecks:                     finalChecks,
	}, nil
}

func (a *Agent) applyRuntimeProjectConfig(resolution runtimeProjectConfigResolution) {
	applyProviderHistoryReductionModeToRuntime(a.Runtime, resolution.providerHistoryReductionMode, resolution.providerHistoryReductionModeSet)
	a.cfg().FinalChecks = resolution.finalChecks
}

func (a *Agent) resolveRuntimeProjectFinalChecks(pc *config.ProjectConfig) (config.FinalChecksConfig, error) {
	if pc != nil && pc.FinalChecks != nil {
		return cloneFinalChecksValue(*pc.FinalChecks), nil
	}

	globalCfg, err := config.LoadConfig()
	if err != nil {
		return config.FinalChecksConfig{}, err
	}
	return cloneFinalChecksValue(globalCfg.FinalChecks), nil
}

func cloneFinalChecksValue(fc config.FinalChecksConfig) config.FinalChecksConfig {
	fc.Commands = append([]string(nil), fc.Commands...)
	return fc
}

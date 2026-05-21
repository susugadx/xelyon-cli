package agent

import "github.com/susugadx/xelyon-cli/internal/config"

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

	if err := a.syncRuntimeProjectFinalChecks(pc); err != nil {
		return err
	}
	if err := syncProviderHistoryReductionModeFromProjectConfig(a.Runtime, pc); err != nil {
		return err
	}

	a.promptManager().InvalidateProjectMap()
	a.refreshProjectPrompt("")
	return nil
}

func (a *Agent) syncRuntimeProjectFinalChecks(pc *config.ProjectConfig) error {
	cfg := a.cfg()
	if pc != nil && pc.FinalChecks != nil {
		cfg.FinalChecks = cloneFinalChecksValue(*pc.FinalChecks)
		return nil
	}

	globalCfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cfg.FinalChecks = cloneFinalChecksValue(globalCfg.FinalChecks)
	return nil
}

func cloneFinalChecksValue(fc config.FinalChecksConfig) config.FinalChecksConfig {
	fc.Commands = append([]string(nil), fc.Commands...)
	return fc
}

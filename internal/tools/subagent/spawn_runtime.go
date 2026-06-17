package subagent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultSubAgentModel = "gpt-5.4-mini"

// spawnConfig は sub-agent 実行直前の確定設定です。
type spawnConfig struct {
	taskType               string
	model                  string
	modelProviderConfigKey string
	cfg                    *config.Config
}

// SpawnRuntimeContext は親 agent の実行時選択を sub-agent 起動へ渡します。
type SpawnRuntimeContext struct {
	CurrentModel string
}

func normalizeTaskType(taskType string) string {
	if !ValidTaskType(taskType) {
		return TaskTypeExplore
	}
	return taskType
}

func prepareSpawnConfigWithRuntimeContext(cfg *config.Config, provider api.Provider, runtimeCtx SpawnRuntimeContext, taskType, model, reasoningEffort string) (*spawnConfig, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	normalizedTaskType := normalizeTaskType(taskType)
	mainProvider := config.CanonicalProviderName(provider.Name())
	mainProviderConfigKey := currentProviderConfigKey(provider)
	subCfg, selection, err := cloneConfigForSubWithRuntimeContext(cfg, mainProvider, mainProviderConfigKey, runtimeCtx, normalizedTaskType, model, reasoningEffort)
	if err != nil {
		return nil, err
	}
	if !subCfg.SubAgent.Enabled {
		return nil, fmt.Errorf("sub-agent is disabled")
	}

	return &spawnConfig{
		taskType:               normalizedTaskType,
		model:                  selection.model,
		modelProviderConfigKey: selection.providerConfigKey,
		cfg:                    subCfg,
	}, nil
}

func applySubAgentPrompt(cfg *config.Config, taskType string, provider api.Provider, model string) {
	if cfg == nil || provider == nil {
		return
	}
	cfg.SubAgentPrompt = PromptForTaskTypeWithConfig(taskType, provider.Name(), model, cfg)
}

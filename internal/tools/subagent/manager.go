package subagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// NewManager は新しい Manager を作成します。
func NewManager() *Manager {
	return NewManagerWithOptions(ManagerOptions{})
}

// NewManagerWithOptions は依存関係を指定して Manager を作成します。
func NewManagerWithOptions(opts ManagerOptions) *Manager {
	manager := &Manager{
		agents: make(map[string]*managedSubAgent),
		runHeadless: func(context.Context, string, string, api.Provider, *config.Config) *RunResult {
			return &RunResult{
				Status:       "error",
				ErrorMessage: "sub-agent runner is not configured",
			}
		},
		providerFactory: api.NewProvider,
		eventCh:         make(chan SubAgentEvent, 256),
	}
	if opts.RunHeadless != nil {
		manager.runHeadless = opts.RunHeadless
	}
	if opts.ProviderFactory != nil {
		manager.providerFactory = opts.ProviderFactory
	}
	return manager
}

// Events はイベントの読み取り専用チャネルを返します。
func (m *Manager) Events() <-chan SubAgentEvent {
	if m == nil {
		return nil
	}
	return m.eventCh
}

// Spawn はサブエージェントを起動し、agent_id を返します。
// ctx がキャンセルされるとサブエージェントの実行も中断されます。
func (m *Manager) Spawn(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config) (string, error) {
	return m.spawnWithRuntimeContext(ctx, message, taskType, model, reasoningEffort, provider, cfg, SpawnRuntimeContext{})
}

// SpawnWithRuntimeContext は親 agent の実行時 model を考慮してサブエージェントを起動します。
func (m *Manager) SpawnWithRuntimeContext(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config, runtimeCtx SpawnRuntimeContext) (string, error) {
	return m.spawnWithRuntimeContext(ctx, message, taskType, model, reasoningEffort, provider, cfg, runtimeCtx)
}

func (m *Manager) spawnWithRuntimeContext(ctx context.Context, message, taskType, model, reasoningEffort string, provider api.Provider, cfg *config.Config, runtimeCtx SpawnRuntimeContext) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("message is required")
	}
	if provider == nil {
		return "", fmt.Errorf("provider is required")
	}
	taskType = normalizeTaskType(taskType)

	spawnCfg, err := prepareSpawnConfigWithRuntimeContext(cfg, provider, runtimeCtx, taskType, model, reasoningEffort)
	if err != nil {
		return "", err
	}

	sub, err := m.allocateRunningAgent(spawnCfg.model, spawnCfg.taskType, spawnCfg.cfg)
	if err != nil {
		return "", err
	}

	subProvider, err := resolveSubProvider(provider, spawnCfg.cfg, spawnCfg.model, spawnCfg.modelProviderConfigKey, m.providerFactory)
	if err != nil {
		m.removeAgent(sub.id)
		return "", err
	}
	applySubAgentPrompt(spawnCfg.cfg, spawnCfg.taskType, subProvider, spawnCfg.model)

	go func() {
		m.runAgent(ctx, sub, message, spawnCfg.model, subProvider, spawnCfg.cfg)
	}()

	return sub.id, nil
}

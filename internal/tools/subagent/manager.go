package subagent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	defaultSubAgentMaxConcurrent = 1
)

// ToolBreakdownEntry はツール別の成功/失敗回数です。
type ToolBreakdownEntry struct {
	Tool     string `json:"tool"`
	Success  int    `json:"success"`
	Failures int    `json:"failures"`
}

// RunResult はサブエージェント実行結果の最小表現です。
type RunResult struct {
	Status             string
	Model              string
	Response           string
	ErrorMessage       string
	InputTokens        int
	CachedTokens       int
	OutputTokens       int
	ThinkingTokens     int
	Cost               float64
	PricingUnavailable bool
	ToolExecutions     int
	ToolBreakdown      []ToolBreakdownEntry
	DurationMs         int64
}

// Runner はサブエージェント実行関数です。
type Runner func(ctx context.Context, message, model string, provider api.Provider, cfg *config.Config) *RunResult

// ProviderFactory はプロバイダー生成関数です。
type ProviderFactory func(providerName string) (api.Provider, error)

// ManagerOptions は Manager の依存関係を注入します。
type ManagerOptions struct {
	RunHeadless     Runner
	ProviderFactory ProviderFactory
}

// Manager はサブエージェントのライフサイクルを管理します。
type Manager struct {
	mu              sync.Mutex
	agents          map[string]*managedSubAgent
	counter         atomic.Int64
	runHeadless     Runner
	providerFactory ProviderFactory
	eventCh         chan SubAgentEvent
}

type managedSubAgent struct {
	id        string
	model     string
	taskType  string
	status    string
	result    *RunResult
	done      chan struct{}
	startTime time.Time
}

// WaitResult は wait_agent の 1 エージェント分の結果です。
type WaitResult struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Output  string `json:"output"`
}

// WaitResponse は wait_agent のレスポンス全体です。
type WaitResponse struct {
	Results []WaitResult `json:"results"`
	Status  string       `json:"status"`
}

// SubAgentStats はサブエージェント 1 件分の統計です。
type SubAgentStats struct {
	ID                 string               `json:"id"`
	Model              string               `json:"model"`
	TaskType           string               `json:"task_type"`
	Status             string               `json:"status"`
	ErrorMessage       string               `json:"error_message"`
	InputTokens        int                  `json:"input_tokens"`
	CachedTokens       int                  `json:"cached_tokens"`
	OutputTokens       int                  `json:"output_tokens"`
	ThinkingTokens     int                  `json:"thinking_tokens"`
	Cost               float64              `json:"cost"`
	PricingUnavailable bool                 `json:"pricing_unavailable,omitempty"`
	ToolExecutions     int                  `json:"tool_executions"`
	ToolBreakdown      []ToolBreakdownEntry `json:"tool_breakdown,omitempty"`
	DurationMs         int64                `json:"duration_ms"`
}

// SubAgentSummary は全サブエージェントの集約統計です。
type SubAgentSummary struct {
	Agents             []SubAgentStats `json:"agents"`
	TotalSpawned       int             `json:"total_spawned"`
	TotalCompleted     int             `json:"total_completed"`
	TotalErrors        int             `json:"total_errors"`
	TotalRunning       int             `json:"total_running"`
	TotalInput         int             `json:"total_input"`
	TotalCached        int             `json:"total_cached"`
	TotalOutput        int             `json:"total_output"`
	TotalThinking      int             `json:"total_thinking"`
	TotalCost          float64         `json:"total_cost"`
	PricingUnavailable bool            `json:"pricing_unavailable,omitempty"`
	TotalTools         int             `json:"total_tools"`
}

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

func (m *Manager) runAgent(ctx context.Context, sub *managedSubAgent, message, model string, provider api.Provider, cfg *config.Config) {
	runCtx := WithEventChannel(ctx, m.eventCh)
	runCtx = WithAgentID(runCtx, sub.id)

	defer close(sub.done)
	defer func() {
		status, result := m.agentOutcome(sub)
		EmitCompletionEvent(runCtx, status, result)
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			m.markAgentPanic(sub, recovered)
		}
	}()

	result := normalizeAgentResult(m.runHeadless(runCtx, message, model, provider, cfg))
	m.setAgentResult(sub, result)
}

func normalizeAgentResult(result *RunResult) *RunResult {
	if result == nil {
		return &RunResult{
			Status:       "error",
			ErrorMessage: "sub-agent runner returned nil result",
		}
	}
	if result.Status == "running" {
		return &RunResult{
			Status:       "error",
			ErrorMessage: "sub-agent runner returned invalid running status",
		}
	}
	return result
}

func (m *Manager) allocateRunningAgent(model, taskType string, cfg *config.Config) (*managedSubAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	limit := maxConcurrent(cfg)
	if m.runningCountLocked() >= limit {
		return nil, fmt.Errorf("max concurrent sub-agents reached (%d)", limit)
	}

	id := fmt.Sprintf("sub-%03d", m.counter.Add(1))
	sub := &managedSubAgent{
		id:        id,
		model:     model,
		taskType:  taskType,
		status:    "running",
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
	m.agents[id] = sub
	return sub, nil
}

func (m *Manager) removeAgent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, id)
}

func (m *Manager) markAgentPanic(sub *managedSubAgent, recovered interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.status = "error"
	sub.result = &RunResult{
		Status:       "error",
		ErrorMessage: fmt.Sprintf("panic: %v", recovered),
	}
}

func (m *Manager) setAgentResult(sub *managedSubAgent, result *RunResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.result = result
	if result.Status == "completed" {
		sub.status = "completed"
		return
	}
	sub.status = "error"
}

func (m *Manager) agentOutcome(sub *managedSubAgent) (string, *RunResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return sub.status, sub.result
}

func (m *Manager) getAgent(id string) (*managedSubAgent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.agents[id]
	return sub, ok
}

// Wait は指定されたサブエージェントの完了を待ちます。
func (m *Manager) Wait(ids []string, timeoutMs int) WaitResponse {
	results := make([]WaitResult, len(ids))
	status := "completed"

	var deadline <-chan time.Time
	if timeoutMs > 0 {
		deadline = time.After(time.Duration(timeoutMs) * time.Millisecond)
	}

	timedOut := false
	for i, id := range ids {
		sub, ok := m.getAgent(id)
		if !ok {
			results[i] = WaitResult{AgentID: id, Status: "error", Output: "agent not found"}
			status = "error"
			continue
		}

		if timedOut {
			results[i] = m.snapshotOrTimeout(sub, true)
			if results[i].Status == "error" {
				status = "error"
			}
			continue
		}

		if deadline == nil {
			<-sub.done
			results[i] = m.snapshotOrTimeout(sub, false)
		} else {
			select {
			case <-sub.done:
				results[i] = m.snapshotOrTimeout(sub, false)
			case <-deadline:
				timedOut = true
				results[i] = m.snapshotOrTimeout(sub, true)
			}
		}

		switch results[i].Status {
		case "error":
			status = "error"
		case "timeout":
			if status != "error" {
				status = "timeout"
			}
		}
	}

	return WaitResponse{
		Results: results,
		Status:  status,
	}
}

// GetSummary は全サブエージェントの統計サマリーを返します。
func (m *Manager) GetSummary() SubAgentSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	summary := SubAgentSummary{
		Agents:       make([]SubAgentStats, 0, len(m.agents)),
		TotalSpawned: len(m.agents),
	}
	if len(m.agents) == 0 {
		return summary
	}

	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		sub := m.agents[id]
		stats := SubAgentStats{
			ID:       sub.id,
			Model:    sub.model,
			TaskType: sub.taskType,
			Status:   sub.status,
		}
		if sub.result != nil {
			if sub.result.Model != "" {
				stats.Model = sub.result.Model
			}
			stats.ErrorMessage = sub.result.ErrorMessage
			stats.InputTokens = sub.result.InputTokens
			stats.CachedTokens = sub.result.CachedTokens
			stats.OutputTokens = sub.result.OutputTokens
			stats.ThinkingTokens = sub.result.ThinkingTokens
			stats.Cost = sub.result.Cost
			stats.PricingUnavailable = sub.result.PricingUnavailable
			stats.ToolExecutions = sub.result.ToolExecutions
			stats.ToolBreakdown = sub.result.ToolBreakdown
			stats.DurationMs = sub.result.DurationMs

			summary.TotalInput += stats.InputTokens
			summary.TotalCached += stats.CachedTokens
			summary.TotalOutput += stats.OutputTokens
			summary.TotalThinking += stats.ThinkingTokens
			summary.TotalCost += stats.Cost
			if stats.PricingUnavailable {
				summary.PricingUnavailable = true
			}
			summary.TotalTools += stats.ToolExecutions
		}

		switch sub.status {
		case "completed":
			summary.TotalCompleted++
		case "error":
			summary.TotalErrors++
		case "running":
			summary.TotalRunning++
		}

		summary.Agents = append(summary.Agents, stats)
	}

	return summary
}

func (m *Manager) runningCountLocked() int {
	count := 0
	for _, sub := range m.agents {
		if sub.status == "running" {
			count++
		}
	}
	return count
}

func (m *Manager) snapshotOrTimeout(sub *managedSubAgent, timeout bool) WaitResult {
	if sub == nil {
		return WaitResult{Status: "error", Output: "agent not found"}
	}

	if timeout {
		select {
		case <-sub.done:
		default:
			return WaitResult{
				AgentID: sub.id,
				Status:  "timeout",
				Output:  "",
			}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	output := ""
	if sub.result != nil {
		switch {
		case sub.result.Status == "error" && sub.result.ErrorMessage != "":
			output = sub.result.ErrorMessage
		case sub.result.Response != "":
			output = sub.result.Response
		case sub.result.ErrorMessage != "":
			output = sub.result.ErrorMessage
		}
	}

	return WaitResult{
		AgentID: sub.id,
		Status:  sub.status,
		Output:  output,
	}
}

func maxConcurrent(cfg *config.Config) int {
	if cfg == nil || cfg.SubAgent.MaxConcurrent <= 0 {
		return defaultSubAgentMaxConcurrent
	}
	return cfg.SubAgent.MaxConcurrent
}

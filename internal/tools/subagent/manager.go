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
	defaultSubAgentModel         = "gpt-5.4-mini"
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
	Status         string
	Model          string
	Response       string
	ErrorMessage   string
	InputTokens    int
	CachedTokens   int
	OutputTokens   int
	ThinkingTokens int
	Cost           float64
	ToolExecutions int
	ToolBreakdown  []ToolBreakdownEntry
	DurationMs     int64
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
	ID             string               `json:"id"`
	Model          string               `json:"model"`
	TaskType       string               `json:"task_type"`
	Status         string               `json:"status"`
	ErrorMessage   string               `json:"error_message"`
	InputTokens    int                  `json:"input_tokens"`
	CachedTokens   int                  `json:"cached_tokens"`
	OutputTokens   int                  `json:"output_tokens"`
	ThinkingTokens int                  `json:"thinking_tokens"`
	Cost           float64              `json:"cost"`
	ToolExecutions int                  `json:"tool_executions"`
	ToolBreakdown  []ToolBreakdownEntry `json:"tool_breakdown,omitempty"`
	DurationMs     int64                `json:"duration_ms"`
}

// SubAgentSummary は全サブエージェントの集約統計です。
type SubAgentSummary struct {
	Agents         []SubAgentStats `json:"agents"`
	TotalSpawned   int             `json:"total_spawned"`
	TotalCompleted int             `json:"total_completed"`
	TotalErrors    int             `json:"total_errors"`
	TotalRunning   int             `json:"total_running"`
	TotalInput     int             `json:"total_input"`
	TotalCached    int             `json:"total_cached"`
	TotalOutput    int             `json:"total_output"`
	TotalThinking  int             `json:"total_thinking"`
	TotalCost      float64         `json:"total_cost"`
	TotalTools     int             `json:"total_tools"`
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
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("message is required")
	}
	if provider == nil {
		return "", fmt.Errorf("provider is required")
	}
	if !ValidTaskType(taskType) {
		taskType = TaskTypeExplore
	}

	mainProvider := config.CanonicalProviderName(provider.Name())
	subCfg, resolvedModel, err := cloneConfigForSub(cfg, mainProvider, taskType, model, reasoningEffort)
	if err != nil {
		return "", err
	}
	if !subCfg.SubAgent.Enabled {
		return "", fmt.Errorf("sub-agent is disabled")
	}

	m.mu.Lock()
	if m.runningCountLocked() >= maxConcurrent(subCfg) {
		m.mu.Unlock()
		return "", fmt.Errorf("max concurrent sub-agents reached (%d)", maxConcurrent(subCfg))
	}

	id := fmt.Sprintf("sub-%03d", m.counter.Add(1))
	sub := &managedSubAgent{
		id:        id,
		model:     resolvedModel,
		taskType:  taskType,
		status:    "running",
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
	m.agents[id] = sub
	m.mu.Unlock()

	subProvider, err := resolveSubProvider(provider, subCfg, resolvedModel, m.providerFactory)
	if err != nil {
		m.mu.Lock()
		delete(m.agents, id)
		m.mu.Unlock()
		return "", err
	}
	subCfg.SubAgentPrompt = PromptForTaskType(taskType, subProvider.Name(), resolvedModel)

	go func() {
		defer close(sub.done)
		defer func() {
			m.mu.Lock()
			status := sub.status
			result := sub.result
			m.mu.Unlock()

			EmitEvent(ctx, SubAgentEvent{
				Tool:    "_completed",
				Phase:   "end",
				Output:  formatCompletionEventOutput(status, result),
				Success: status == "completed",
			})
		}()
		defer func() {
			if recovered := recover(); recovered != nil {
				m.mu.Lock()
				sub.status = "error"
				sub.result = &RunResult{
					Status:       "error",
					ErrorMessage: fmt.Sprintf("panic: %v", recovered),
				}
				m.mu.Unlock()
			}
		}()

		ctx = WithEventChannel(ctx, m.eventCh)
		ctx = WithAgentID(ctx, id)

		result := m.runHeadless(ctx, message, resolvedModel, subProvider, subCfg)
		if result == nil {
			result = &RunResult{
				Status:       "error",
				ErrorMessage: "sub-agent runner returned nil result",
			}
		}

		m.mu.Lock()
		sub.result = result
		switch result.Status {
		case "completed":
			sub.status = "completed"
		case "running":
			sub.status = "error"
			sub.result = &RunResult{
				Status:       "error",
				ErrorMessage: "sub-agent runner returned invalid running status",
			}
		default:
			sub.status = "error"
		}
		m.mu.Unlock()
	}()

	return id, nil
}

func formatCompletionEventOutput(status string, result *RunResult) string {
	if result == nil {
		if status == "" {
			return "unknown"
		}
		return status
	}
	if status != "completed" {
		if result.ErrorMessage != "" {
			return result.ErrorMessage
		}
		if status != "" {
			return status
		}
		return "error"
	}

	parts := make([]string, 0, 2)
	if result.ToolExecutions > 0 {
		label := "tools"
		if result.ToolExecutions == 1 {
			label = "tool"
		}
		parts = append(parts, fmt.Sprintf("%d %s", result.ToolExecutions, label))
	}
	if result.DurationMs > 0 {
		parts = append(parts, formatEventDuration(result.DurationMs))
	}
	if len(parts) == 0 {
		return "completed"
	}
	return fmt.Sprintf("completed (%s)", strings.Join(parts, ", "))
}

func formatEventDuration(durationMs int64) string {
	if durationMs < 1000 {
		return fmt.Sprintf("%dms", durationMs)
	}
	seconds := float64(durationMs) / 1000
	if durationMs%1000 == 0 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	return fmt.Sprintf("%.1fs", seconds)
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
		m.mu.Lock()
		sub, ok := m.agents[id]
		m.mu.Unlock()
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
			stats.ToolExecutions = sub.result.ToolExecutions
			stats.ToolBreakdown = sub.result.ToolBreakdown
			stats.DurationMs = sub.result.DurationMs

			summary.TotalInput += stats.InputTokens
			summary.TotalCached += stats.CachedTokens
			summary.TotalOutput += stats.OutputTokens
			summary.TotalThinking += stats.ThinkingTokens
			summary.TotalCost += stats.Cost
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

func cloneConfigForSub(cfg *config.Config, mainProvider, taskType, model, reasoningEffort string) (*config.Config, string, error) {
	cloned := config.CloneConfig(cfg)
	if cloned == nil {
		cloned = config.DefaultConfig()
	}
	if !ValidTaskType(taskType) {
		taskType = TaskTypeExplore
	}

	resolvedModel := normalizeSubAgentModel(model)
	if resolvedModel == "" {
		resolvedModel = normalizeSubAgentModel(cloned.SubAgent.DefaultModel)
	}
	if resolvedModel == "" {
		resolvedModel = inferSubAgentModel(mainProvider)
	}
	if resolvedModel == "" {
		resolvedModel = defaultSubAgentModel
	}

	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = DefaultEffortForTaskType(taskType)
	}
	if effort == "" {
		effort = strings.TrimSpace(cloned.SubAgent.DefaultEffort)
	}

	if err := applyReasoningEffort(cloned, effort); err != nil {
		return nil, "", err
	}

	return cloned, resolvedModel, nil
}

func normalizeSubAgentModel(model string) string {
	model = strings.TrimSpace(model)
	switch strings.ToLower(model) {
	case "", "sub_agent.default_model":
		return ""
	default:
		return model
	}
}

func inferSubAgentModel(provider string) string {
	switch config.CanonicalProviderName(provider) {
	case "openai":
		return "gpt-5.4-mini"
	case "claude":
		return "claude-haiku-4-5-20251001"
	case "gemini":
		return "gemini-3.1-flash-lite-preview"
	case "deepseek":
		return "deepseek-chat"
	case "groq":
		return "llama-3.3-70b-versatile"
	case "openrouter":
		return "openai/gpt-5.4-mini"
	default:
		return ""
	}
}

func applyReasoningEffort(cfg *config.Config, effort string) error {
	switch normalizeReasoningEffort(effort) {
	case "", "off":
		cfg.Thinking.Enabled = false
		return nil
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = normalizeReasoningEffort(effort)
		return nil
	default:
		return fmt.Errorf("invalid reasoning_effort: %s", effort)
	}
}

func normalizeReasoningEffort(effort string) string {
	return strings.ToLower(strings.TrimSpace(effort))
}

type providerConfigKeyProvider interface {
	ProviderConfigKey() string
}

func currentProviderConfigKey(current api.Provider) string {
	if current == nil {
		return ""
	}
	if keyed, ok := current.(providerConfigKeyProvider); ok {
		if key := config.ActiveProviderConfigKey(keyed.ProviderConfigKey()); key != "" {
			return key
		}
	}
	return config.ActiveProviderConfigKey(current.Name())
}

func resolveSubProvider(current api.Provider, cfg *config.Config, model string, factory ProviderFactory) (api.Provider, error) {
	if current == nil {
		return nil, fmt.Errorf("provider is required")
	}

	currentName := config.CanonicalProviderName(current.Name())
	currentConfigKey := currentProviderConfigKey(current)
	target := currentName
	if cfg != nil {
		target = cfg.ResolveProviderForModel(currentName, model)
	}
	if target == "" {
		target = currentName
	}
	factoryProviderName := target
	if currentConfigKey != "" && config.SameProviderRuntimeIdentity(currentConfigKey, target) {
		factoryProviderName = currentConfigKey
	}
	if factory == nil {
		factory = api.NewProvider
	}
	provider, err := factory(factoryProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", factoryProviderName, err)
	}
	resetSubProviderState(provider)
	return provider, nil
}

func resetSubProviderState(provider api.Provider) {
	if provider == nil {
		return
	}
	if cacheClearable, ok := provider.(api.CacheClearable); ok {
		cacheClearable.ClearCache()
		return
	}
	if responseIDSetter, ok := provider.(interface{ SetResponseID(string) }); ok {
		responseIDSetter.SetResponseID("")
	}
}

func maxConcurrent(cfg *config.Config) int {
	if cfg == nil || cfg.SubAgent.MaxConcurrent <= 0 {
		return defaultSubAgentMaxConcurrent
	}
	return cfg.SubAgent.MaxConcurrent
}

package subagent

import (
	"context"
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
	AgentID       string               `json:"agent_id"`
	Status        string               `json:"status"`
	Output        string               `json:"output"`
	ToolBreakdown []ToolBreakdownEntry `json:"tool_breakdown,omitempty"`
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

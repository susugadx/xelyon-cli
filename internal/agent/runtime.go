package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RuntimeOptions は一時的な内部 runtime gate を束ねる。
type RuntimeOptions struct {
	EnableCurrentTaskStateContext         bool
	EnableProviderHistoryRehydrateContext bool
	EnableProviderHistoryReduction        bool
	ProviderHistoryReductionMode          ProviderHistoryReductionMode
	ProviderHistoryReductionModeSet       bool
}

// AgentRuntime は agent/session 単位で保持する実行時 state を束ねる。
type AgentRuntime struct {
	Registry                            *tools.Registry
	ToolCache                           *ToolCache
	Config                              *config.Config
	Options                             RuntimeOptions
	ProjectConfig                       *ProjectConfigStore
	InvocationCWD                       string
	AutoApprove                         bool
	UI                                  *ui.Runtime
	AuditLogger                         audit.ToolLogger
	SubAgentManager                     *subagent.Manager
	TaskLedger                          *taskstate.Store
	LastProviderHistoryProjectionReport ProviderHistoryProjectionReport

	managedTaskLedger       *taskstate.Store
	taskLedgerInvocationCWD string
}

// NewAgentRuntime は CLI 互換の初期値を持つ runtime を返す。
func NewAgentRuntime() *AgentRuntime {
	return normalizeAgentRuntime(nil)
}

// NewAgentRuntimeWithConfig は指定設定を保持した runtime を返す。
func NewAgentRuntimeWithConfig(cfg *config.Config) *AgentRuntime {
	return normalizeAgentRuntime(&AgentRuntime{Config: cfg})
}

func normalizeAgentRuntime(runtime *AgentRuntime) *AgentRuntime {
	if runtime == nil {
		runtime = &AgentRuntime{}
	}
	if runtime.Registry == nil {
		runtime.Registry = tools.DefaultRegistry.Clone()
	}
	if runtime.ToolCache == nil {
		cache := NewToolCache()
		_ = cache.Load()
		runtime.ToolCache = cache
	}
	if runtime.Config == nil {
		runtime.Config = config.CloneConfig(config.DefaultConfig())
	} else {
		runtime.Config = config.CloneConfig(runtime.Config)
	}
	if runtime.ProjectConfig == nil {
		runtime.ProjectConfig = NewProjectConfigStore()
	}
	if runtime.InvocationCWD == "" {
		runtime.InvocationCWD = resolveRuntimeInvocationCWD()
	}
	if runtime.UI == nil {
		runtime.UI = ui.NewRuntime(nil, nil, nil)
	}
	if runtime.AuditLogger == nil {
		runtime.AuditLogger = audit.NewDisabledLogger()
	}
	if runtime.SubAgentManager == nil {
		runtime.SubAgentManager = newSubAgentManager()
	}
	runtime.ensureTaskLedger()
	registerRuntimeSubAgentTools(runtime)
	return runtime
}

// SetConfig は runtime の設定をディープコピーして差し替える。
func (r *AgentRuntime) SetConfig(cfg *config.Config) {
	if r == nil {
		return
	}
	r.Config = config.CloneConfig(cfg)
}

func (r *AgentRuntime) effectiveRegistry() *tools.Registry {
	if r == nil || r.Registry == nil {
		return tools.DefaultRegistry
	}
	return r.Registry
}

func (r *AgentRuntime) effectiveToolCache() *ToolCache {
	if r == nil {
		return nil
	}
	return r.ToolCache
}

func (r *AgentRuntime) effectiveConfig() *config.Config {
	if r == nil || r.Config == nil {
		return config.DefaultConfig()
	}
	return r.Config
}

func (r *AgentRuntime) effectiveProjectConfigStore() *ProjectConfigStore {
	if r == nil || r.ProjectConfig == nil {
		return defaultProjectConfigStore
	}
	return r.ProjectConfig
}

func (r *AgentRuntime) effectiveUI() *ui.Runtime {
	if r == nil || r.UI == nil {
		return ui.NewRuntime(nil, nil, nil)
	}
	return r.UI
}

func (r *AgentRuntime) effectiveAuditLogger() audit.ToolLogger {
	if r == nil || r.AuditLogger == nil {
		return audit.NewDisabledLogger()
	}
	return r.AuditLogger
}

func (r *AgentRuntime) effectiveAutoApprove() bool {
	if r == nil {
		return false
	}
	return r.AutoApprove
}

func newSubAgentManager() *subagent.Manager {
	return subagent.NewManagerWithOptions(subagent.ManagerOptions{
		RunHeadless: func(ctx context.Context, message, model string, provider api.Provider, cfg *config.Config) *subagent.RunResult {
			result := RunHeadlessWithConfig(ctx, message, model, provider, cfg)
			if result == nil {
				return &subagent.RunResult{
					Status:       "error",
					ErrorMessage: "headless run returned nil result",
				}
			}
			if result.Status == "success" {
				return headlessResultToSubAgentResult("completed", result)
			}
			errorMessage := ""
			if result.Error != nil {
				errorMessage = result.Error.Message
			}
			subResult := headlessResultToSubAgentResult("error", result)
			subResult.ErrorMessage = errorMessage
			return subResult
		},
	})
}

func headlessResultToSubAgentResult(status string, result *HeadlessResult) *subagent.RunResult {
	subResult := &subagent.RunResult{
		Status:             status,
		Model:              result.Model,
		Response:           result.Response,
		Cost:               result.Cost,
		PricingUnavailable: result.PricingUnavailable,
		ToolExecutions:     len(result.ToolCalls),
		DurationMs:         result.DurationMs,
	}
	if result.Tokens != nil {
		subResult.InputTokens = result.Tokens.Input
		subResult.CachedTokens = result.Tokens.Cached
		subResult.OutputTokens = result.Tokens.Output
		subResult.ThinkingTokens = result.Tokens.Thinking
	}

	// ToolCalls からツール別の成功/失敗回数を集計
	if len(result.ToolCalls) > 0 {
		type toolCount struct {
			success  int
			failures int
		}
		counts := make(map[string]*toolCount)
		order := make([]string, 0)
		for _, tc := range result.ToolCalls {
			c, exists := counts[tc.Tool]
			if !exists {
				c = &toolCount{}
				counts[tc.Tool] = c
				order = append(order, tc.Tool)
			}
			if tc.Success {
				c.success++
			} else {
				c.failures++
			}
		}
		breakdown := make([]subagent.ToolBreakdownEntry, 0, len(order))
		for _, tool := range order {
			c := counts[tool]
			breakdown = append(breakdown, subagent.ToolBreakdownEntry{
				Tool:     tool,
				Success:  c.success,
				Failures: c.failures,
			})
		}
		subResult.ToolBreakdown = breakdown
	}

	return subResult
}

func registerRuntimeSubAgentTools(runtime *AgentRuntime) {
	if runtime == nil || runtime.Registry == nil || runtime.SubAgentManager == nil || runtime.Config == nil {
		return
	}
	if !runtime.Config.SubAgent.Enabled {
		return
	}
	if !runtime.Registry.HasTool("spawn_agent") {
		runtime.Registry.Register(subagent.NewSpawnAgentTool(runtime.SubAgentManager))
	}
	if !runtime.Registry.HasTool("wait_agent") {
		runtime.Registry.Register(subagent.NewWaitAgentTool(runtime.SubAgentManager))
	}
}

func (a *Agent) cfg() *config.Config {
	if a == nil || a.Runtime == nil {
		return config.DefaultConfig()
	}
	return a.Runtime.effectiveConfig()
}

func (a *Agent) registry() *tools.Registry {
	if a == nil || a.Runtime == nil {
		return tools.DefaultRegistry
	}
	return a.Runtime.effectiveRegistry()
}

func (a *Agent) subAgentManager() *subagent.Manager {
	if a == nil || a.Runtime == nil {
		return nil
	}
	return a.Runtime.SubAgentManager
}

func (a *Agent) autoApprove() bool {
	if a == nil {
		return false
	}
	if a.Runtime != nil {
		return a.Runtime.effectiveAutoApprove()
	}
	return a.AutoApprove
}

func (a *Agent) ui() *ui.Runtime {
	if a == nil || a.Runtime == nil {
		return ui.NewRuntime(nil, nil, nil)
	}
	return a.Runtime.effectiveUI()
}

func (a *Agent) auditLogger() audit.ToolLogger {
	if a == nil || a.Runtime == nil {
		return audit.NewDisabledLogger()
	}
	return a.Runtime.effectiveAuditLogger()
}

func (a *Agent) setAutoApprove(enabled bool) {
	if a == nil {
		return
	}
	a.AutoApprove = enabled
	if a.Runtime != nil {
		a.Runtime.AutoApprove = enabled
	}
}

func (a *Agent) setRuntimeConfig(cfg *config.Config) {
	if a == nil {
		return
	}
	if a.Runtime == nil {
		a.Runtime = NewAgentRuntimeWithConfig(cfg)
		return
	}
	a.Runtime.SetConfig(cfg)
}

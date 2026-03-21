package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// AgentRuntime は agent/session 単位で保持する実行時 state を束ねる。
type AgentRuntime struct {
	Registry        *tools.Registry
	ToolCache       *ToolCache
	Config          *config.Config
	AutoApprove     bool
	UI              *ui.Runtime
	AuditLogger     audit.ToolLogger
	SubAgentManager *subagent.Manager
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
	if runtime.UI == nil {
		runtime.UI = ui.NewRuntime(nil, nil, nil)
	}
	if runtime.AuditLogger == nil {
		runtime.AuditLogger = audit.NewDisabledLogger()
	}
	if runtime.SubAgentManager == nil {
		runtime.SubAgentManager = newSubAgentManager()
	}
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
		RunHeadless: func(message, model string, provider api.Provider, cfg *config.Config) *subagent.RunResult {
			result := RunHeadlessWithConfig(message, model, provider, cfg)
			if result == nil {
				return &subagent.RunResult{
					Status:       "error",
					ErrorMessage: "headless run returned nil result",
				}
			}
			if result.Status == "success" {
				return &subagent.RunResult{
					Status:   "completed",
					Response: result.Response,
				}
			}
			errorMessage := ""
			if result.Error != nil {
				errorMessage = result.Error.Message
			}
			return &subagent.RunResult{
				Status:       "error",
				Response:     result.Response,
				ErrorMessage: errorMessage,
			}
		},
	})
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

func (a *Agent) requestContext(ctx context.Context) context.Context {
	ctx = tools.WithRegistry(ctx, a.registry())
	ctx = tools.WithConfig(ctx, a.cfg())
	ctx = ui.WithRuntime(ctx, a.ui())
	return ctx
}

func (a *Agent) parseToolCalls(response string) []*tools.ToolCall {
	return tools.ParseToolCallsWithRegistry(response, a.registry(), a.ui().ErrorOutput())
}

func (a *Agent) estimateToolDefinitionTokens() int {
	total := 0
	for _, def := range a.registry().GetToolDefinitions() {
		total += token.EstimateStructuredValueTokenCountForModel(a.CurrentModel, def)
	}
	return total
}

func (a *Agent) countToolsByType() (builtin, mcp int) {
	for _, def := range a.registry().GetToolDefinitions() {
		if strings.HasPrefix(def.Name, "mcp_") {
			mcp++
		} else {
			builtin++
		}
	}
	return
}

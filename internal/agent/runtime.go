package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// AgentRuntime は agent/session 単位で保持する実行時 state を束ねる。
type AgentRuntime struct {
	Registry    *tools.Registry
	ToolCache   *ToolCache
	Config      *config.Config
	AutoApprove bool
}

// NewAgentRuntime は CLI 互換の初期値を持つ runtime を返す。
func NewAgentRuntime() *AgentRuntime {
	return normalizeAgentRuntime(nil)
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
		runtime.Config = config.CloneConfig(config.GetGlobalConfig())
	} else {
		runtime.Config = config.CloneConfig(runtime.Config)
	}
	return runtime
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
		return config.GetGlobalConfig()
	}
	return r.Config
}

func (r *AgentRuntime) effectiveAutoApprove() bool {
	if r == nil {
		return false
	}
	return r.AutoApprove
}

func (a *Agent) cfg() *config.Config {
	if a == nil || a.Runtime == nil {
		return config.GetGlobalConfig()
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

func (a *Agent) setAutoApprove(enabled bool) {
	if a == nil {
		return
	}
	a.AutoApprove = enabled
	if a.Runtime != nil {
		a.Runtime.AutoApprove = enabled
	}
}

func (a *Agent) requestContext(ctx context.Context) context.Context {
	ctx = tools.WithRegistry(ctx, a.registry())
	ctx = tools.WithConfig(ctx, a.cfg())
	return ctx
}

func (a *Agent) parseToolCalls(response string) []*tools.ToolCall {
	return tools.ParseToolCallsWithRegistry(response, a.registry())
}

func (a *Agent) estimateToolDefinitionTokens() int {
	total := 0
	for _, def := range a.registry().GetToolDefinitions() {
		jsonBytes, _ := json.Marshal(def)
		total += token.EstimateTokenCount(string(jsonBytes))
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

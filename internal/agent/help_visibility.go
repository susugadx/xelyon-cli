package agent

import "github.com/susugadx/xelyon-cli/internal/tools"

func helpToolVisibilityPolicy(agent *Agent) toolVisibilityPolicy {
	if agent == nil {
		return newToolVisibilityPolicy(EditToolModeApplyPatch, toolSurfacePhaseNormal, toolVisibilityOptions{allowSubAgents: true})
	}
	return agent.toolVisibilityPolicy(helpToolSurfacePhase(agent), toolVisibilityOptions{allowSubAgents: true})
}

func helpToolSurfacePhase(agent *Agent) toolSurfacePhase {
	if agent == nil {
		return toolSurfacePhaseNormal
	}
	return agent.currentToolSurfacePhase()
}

// helpToolSectionsForCurrentRuntime resolves the currently visible tool
// definitions from the runtime registry/policy boundary, then hands off summary
// construction to the help summary owner.
func helpToolSectionsForCurrentRuntime(agent *Agent, policy toolVisibilityPolicy) helpSections {
	builtInDefs := make(map[string]tools.ToolDefinition)
	mcpDefs := make(map[string]tools.ToolDefinition)
	for _, def := range helpVisibleToolDefinitions(agent, policy) {
		if isMCPToolDefinition(def) {
			mcpDefs[def.Name] = def
			continue
		}
		builtInDefs[def.Name] = def
	}

	return helpSections{
		builtIn: buildOrderedBuiltInHelpSummaries(builtInDefs, policy.investigationSurface),
		mcp:     buildSortedMCPHelpSummaries(mcpDefs),
	}
}

// helpVisibleToolDefinitions returns the current live `/help` tool set.
// It combines current-surface exclusions owned by tool visibility policy with
// any additional runtime exclusions already applied to the registry.
func helpVisibleToolDefinitions(agent *Agent, policy toolVisibilityPolicy) []tools.ToolDefinition {
	registry := tools.DefaultRegistry
	if agent != nil {
		registry = agent.registry()
	}

	effectiveRegistry := registry.Clone()
	effectiveRegistry.SetExcludedTools(mergeSurfaceManagedExcludedTools(registry.GetExcludedTools(), policy))
	return effectiveRegistry.GetToolDefinitions()
}

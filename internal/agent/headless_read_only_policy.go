package agent

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/tools"
	toolskills "github.com/susugadx/xelyon-cli/internal/tools/skills"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
)

func newHeadlessAgentRuntime(cfg *config.Config, options HeadlessRunOptions) *AgentRuntime {
	runtime := &AgentRuntime{
		Config: headlessRuntimeConfigForOptions(cfg, options),
	}
	if options.ReadOnly {
		runtime.Options.ReadOnly = true
		runtime.ToolCache = NewEphemeralToolCache()
		runtime.Options.SkipDevArtifactCleanup = true
	}
	return normalizeAgentRuntime(runtime)
}

func headlessRuntimeConfigForOptions(cfg *config.Config, options HeadlessRunOptions) *config.Config {
	if !options.ReadOnly {
		return cfg
	}
	runtimeCfg := config.CloneConfig(cfg)
	runtimeCfg.MCP.Headless = false
	runtimeCfg.ProjectMap.Enabled = false
	runtimeCfg.LSP.Enabled = false
	return runtimeCfg
}

func headlessReadOnlyExcludedTools(registry *tools.Registry, baseExcluded []string) []string {
	excluded := append([]string(nil), baseExcluded...)
	excluded = appendHeadlessReadOnlyDeniedToolNames(excluded,
		"bash",
		toolskills.RunSkillScriptToolName,
		subagent.SpawnAgentToolName,
		subagent.WaitAgentToolName,
	)
	if registry == nil {
		return excluded
	}
	for _, def := range registry.GetToolDefinitions() {
		excluded = appendHeadlessReadOnlyDeniedToolNames(excluded, def.Name)
	}
	return excluded
}

func appendHeadlessReadOnlyDeniedToolNames(excluded []string, toolNames ...string) []string {
	for _, toolName := range toolNames {
		if _, denied := classifyHeadlessReadOnlyDeniedToolName(toolName); denied {
			excluded = appendUniqueStrings(excluded, toolName)
		}
	}
	return excluded
}

func headlessReadOnlyMCPXMLToolAttempts(response string) []*tools.ToolCall {
	var calls []*tools.ToolCall
	for _, tagName := range tools.XMLToolCallCandidateTagNames(response) {
		if !mcpnames.IsExportedToolName(tagName) {
			continue
		}
		calls = append(calls, &tools.ToolCall{
			Tool: tagName,
			Args: map[string]string{},
		})
	}
	return calls
}

func (r *headlessRunner) readOnlyDeniedToolResult(tc *tools.ToolCall) (tools.ExecutionResult, bool) {
	if r == nil || !r.options.ReadOnly || tc == nil {
		return tools.ExecutionResult{}, false
	}
	output, denied := classifyHeadlessReadOnlyDeniedToolName(tc.Tool)
	if !denied {
		return tools.ExecutionResult{}, false
	}
	return newHeadlessReadOnlyDeniedExecutionResult(output), true
}

func classifyHeadlessReadOnlyDeniedToolName(toolName string) (string, bool) {
	switch toolName {
	case "bash":
		return "Error: read-only mode denied bash tool", true
	case toolskills.RunSkillScriptToolName:
		return "Error: read-only mode denied skill script execution tool: run_skill_script", true
	case subagent.SpawnAgentToolName, subagent.WaitAgentToolName:
		return fmt.Sprintf("Error: read-only mode denied sub-agent tool: %s", toolName), true
	}
	if tools.IsWriteTool(toolName) {
		return fmt.Sprintf("Error: read-only mode denied write-capable tool: %s", toolName), true
	}
	if mcpnames.IsExportedToolName(toolName) {
		return fmt.Sprintf("Error: read-only mode denied MCP tool: %s", toolName), true
	}
	return "", false
}

func newHeadlessReadOnlyDeniedExecutionResult(output string) tools.ExecutionResult {
	return tools.ExecutionResult{
		Result:    output,
		StartedAt: time.Now(),
		Error:     true,
	}
}

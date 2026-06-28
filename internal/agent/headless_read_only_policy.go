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
	excluded = appendUniqueStrings(excluded, "bash", toolskills.RunSkillScriptToolName, subagent.SpawnAgentToolName, subagent.WaitAgentToolName)
	if registry == nil {
		return excluded
	}
	for _, def := range registry.GetToolDefinitions() {
		if isHeadlessReadOnlyDeniedToolName(def.Name) {
			excluded = appendUniqueStrings(excluded, def.Name)
		}
	}
	return excluded
}

func isHeadlessReadOnlyDeniedToolName(toolName string) bool {
	if isHeadlessReadOnlyShellExecutionToolName(toolName) {
		return true
	}
	if tools.IsWriteTool(toolName) || mcpnames.IsExportedToolName(toolName) {
		return true
	}
	return isHeadlessReadOnlySubAgentToolName(toolName)
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
	if tools.IsWriteTool(tc.Tool) {
		return newHeadlessReadOnlyDeniedExecutionResult(fmt.Sprintf("Error: read-only mode denied write-capable tool: %s", tc.Tool)), true
	}
	if mcpnames.IsExportedToolName(tc.Tool) {
		return newHeadlessReadOnlyDeniedExecutionResult(fmt.Sprintf("Error: read-only mode denied MCP tool: %s", tc.Tool)), true
	}
	if isHeadlessReadOnlySubAgentToolName(tc.Tool) {
		return newHeadlessReadOnlyDeniedExecutionResult(fmt.Sprintf("Error: read-only mode denied sub-agent tool: %s", tc.Tool)), true
	}
	if tc.Tool == "bash" {
		return newHeadlessReadOnlyDeniedExecutionResult("Error: read-only mode denied bash tool"), true
	}
	if tc.Tool == toolskills.RunSkillScriptToolName {
		return newHeadlessReadOnlyDeniedExecutionResult("Error: read-only mode denied skill script execution tool: run_skill_script"), true
	}
	return tools.ExecutionResult{}, false
}

func isHeadlessReadOnlyShellExecutionToolName(toolName string) bool {
	switch toolName {
	case "bash", toolskills.RunSkillScriptToolName:
		return true
	default:
		return false
	}
}

func isHeadlessReadOnlySubAgentToolName(toolName string) bool {
	switch toolName {
	case subagent.SpawnAgentToolName, subagent.WaitAgentToolName:
		return true
	default:
		return false
	}
}

func newHeadlessReadOnlyDeniedExecutionResult(output string) tools.ExecutionResult {
	return tools.ExecutionResult{
		Result:    output,
		StartedAt: time.Now(),
		Error:     true,
	}
}

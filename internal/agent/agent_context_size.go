package agent

import (
	"bytes"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
)

// printContextSize はコンテキストサイズをツリー形式で表示する。
func printContextSize(agent *Agent) {
	out := agent.output()
	dim.Fprint(out, buildContextSizeBlock(agent))
}

func buildContextSizeBlock(agent *Agent) string {
	systemPromptTokens := agent.EstimateSystemPromptTokens()
	basePromptTokens := systemPromptTokens
	toolsTokens := 0
	projectMapTokens := estimateProjectMapTokens(agent.SystemPrompt)

	if agent.CurrentProvider != nil && agent.CurrentProvider.IsFunctionCallingEnabled() {
		toolsTokens = agent.estimateToolDefinitionTokens()
	}

	builtinCount, mcpCount := agent.countToolsByType()
	projectTokens := estimateProjectConfigTokens(agent.loadProjectConfig())
	basePromptTokens -= projectMapTokens + projectTokens
	if basePromptTokens < 0 {
		basePromptTokens = 0
	}

	total := systemPromptTokens + toolsTokens

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "📋 Context size: ~%s tok\n", FormatTokens(total))

	lines := []string{
		fmt.Sprintf("Base prompt: ~%s", FormatTokens(basePromptTokens)),
	}
	if mcpCount > 0 {
		lines = append(lines, fmt.Sprintf("Tools (%d+%d MCP): ~%s",
			builtinCount, mcpCount, FormatTokens(toolsTokens)))
	} else {
		lines = append(lines, fmt.Sprintf("Tools (%d): ~%s",
			builtinCount, FormatTokens(toolsTokens)))
	}
	if projectMapTokens > 0 && agent.projectMapFileCount > 0 {
		if agent.projectMapSymbolCount > 0 {
			lines = append(lines, fmt.Sprintf("Project map (%d symbols, %d files): ~%s",
				agent.projectMapSymbolCount, agent.projectMapFileCount, FormatTokens(projectMapTokens)))
		} else {
			lines = append(lines, fmt.Sprintf("Project map (%d files): ~%s",
				agent.projectMapFileCount, FormatTokens(projectMapTokens)))
		}
	}
	if projectTokens > 0 {
		lines = append(lines, fmt.Sprintf("xelyon.yaml: ~%s", FormatTokens(projectTokens)))
	}

	for i, line := range lines {
		connector := "├──"
		if i == len(lines)-1 {
			connector = "└──"
		}
		fmt.Fprintf(&buf, "   %s %s\n", connector, line)
	}

	return buf.String()
}

func estimateProjectMapTokens(systemPrompt string) int {
	section := extractProjectMapSection(systemPrompt)
	if section == "" {
		return 0
	}
	return token.EstimateTokenCount(section)
}

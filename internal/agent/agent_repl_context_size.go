package agent

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/token"
)

// printContextSize はコンテキストサイズをツリー形式で表示
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
	projectTokens := estimateProjectInstructionTokens(agent.loadProjectInstructionBundleCached(false))
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
		line := fmt.Sprintf("Project instructions: ~%s", FormatTokens(projectTokens))
		if labels := projectInstructionStatusLabels(agent.projectInstructionBundleIfLoaded()); labels != "" {
			line += " (" + labels + ")"
		}
		lines = append(lines, line)
	} else if labels := projectInstructionStatusLabels(agent.projectInstructionBundleIfLoaded()); labels != "" {
		lines = append(lines, "Project instructions: ~0 ("+labels+")")
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

func projectInstructionStatusLabels(bundle *config.ProjectInstructionBundle) string {
	if bundle == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if labels := joinInstructionLabels(bundle.ProjectGuidance, bundle.ProjectGuidanceStatus); labels != "" {
		parts = append(parts, "project: "+labels)
	}
	if labels := joinInstructionLabels(bundle.GlobalGuidance, bundle.GlobalGuidanceStatus); labels != "" {
		parts = append(parts, "global: "+labels)
	}
	return strings.Join(parts, "; ")
}

func estimateProjectMapTokens(systemPrompt string) int {
	section := extractProjectMapSection(systemPrompt)
	if section == "" {
		return 0
	}
	return token.EstimateTokenCount(section)
}

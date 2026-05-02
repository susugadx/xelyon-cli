package agent

import (
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/investigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// helpToolSummary is the one-line `/help` representation of a visible tool.
type helpToolSummary struct {
	Name        string
	Description string
}

// helpSections is the rendered grouping contract for `/help`.
// Built-in tools follow the current investigation surface; MCP tools follow the
// connected runtime registry.
type helpSections struct {
	builtIn []helpToolSummary
	mcp     []helpToolSummary
}

var helpToolDisplayOrder = []string{
	investigation.ToolGatherContext,
	investigation.ToolSearchCode,
	investigation.ToolReadFile,
	investigation.ToolListDir,
	"web_search",
	"activate_skill",
	"ask_user_question",
	"apply_patch",
	"str_replace",
	"write_file",
	"delete_file",
	"bash",
	"spawn_agent",
	"wait_agent",
}

func buildOrderedBuiltInHelpSummaries(visibleDefs map[string]tools.ToolDefinition, surface investigation.Surface) []helpToolSummary {
	summaries := make([]helpToolSummary, 0, len(visibleDefs))
	used := make(map[string]struct{}, len(visibleDefs))
	for _, name := range helpToolDisplayOrder {
		def, ok := visibleDefs[name]
		if !ok {
			continue
		}
		summaries = append(summaries, helpToolSummary{
			Name:        name,
			Description: helpToolDescription(name, surface, def.Description),
		})
		used[name] = struct{}{}
	}

	var extras []string
	for name := range visibleDefs {
		if _, ok := used[name]; ok {
			continue
		}
		extras = append(extras, name)
	}
	sort.Strings(extras)
	for _, name := range extras {
		summaries = append(summaries, helpToolSummary{
			Name:        name,
			Description: fallbackHelpDescription(visibleDefs[name].Description),
		})
	}

	return summaries
}

func buildSortedMCPHelpSummaries(visibleDefs map[string]tools.ToolDefinition) []helpToolSummary {
	if len(visibleDefs) == 0 {
		return nil
	}

	names := make([]string, 0, len(visibleDefs))
	for name := range visibleDefs {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]helpToolSummary, 0, len(names))
	for _, name := range names {
		summaries = append(summaries, helpToolSummary{
			Name:        name,
			Description: fallbackHelpDescription(visibleDefs[name].Description),
		})
	}
	return summaries
}

func helpToolDescription(name string, surface investigation.Surface, description string) string {
	if summary, ok := surface.HelpSummary(name); ok {
		return summary
	}

	switch name {
	case "web_search":
		return "Search the web and return summarized findings with source URLs"
	case "activate_skill":
		return "Load full SKILL.md content for one discovered skill on demand"
	case "ask_user_question":
		return "Ask the user a clarification question during plan investigation"
	case "apply_patch":
		return "Primary edit tool for precise patch-based file changes"
	case "str_replace":
		return "Precise same-file replacements from actual tool output"
	case "write_file":
		return "Legacy edit tool to create or overwrite a file"
	case "delete_file":
		return "Legacy edit tool to delete a file"
	case "bash":
		return "Execute shell commands for build, test, git, and local tooling"
	case "spawn_agent":
		return "Spawn a sub-agent for explore/edit/verify tasks"
	case "wait_agent":
		return "Wait for sub-agents to complete"
	default:
		return fallbackHelpDescription(description)
	}
}

func fallbackHelpDescription(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return "Built-in tool"
	}
	for _, paragraph := range strings.Split(desc, "\n\n") {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			continue
		}
		trimmed = normalizeHelpDescriptionLine(strings.TrimLeft(trimmed, "# "))
		if idx := strings.Index(trimmed, ". "); idx >= 0 {
			return strings.TrimSpace(trimmed[:idx+1])
		}
		return trimmed
	}
	return "Built-in tool"
}

func normalizeHelpDescriptionLine(description string) string {
	return strings.Join(strings.Fields(description), " ")
}

func helpSurfaceSummary(surface investigation.Surface) string {
	return surface.Summary()
}

func isMCPToolDefinition(def tools.ToolDefinition) bool {
	return strings.HasPrefix(def.Name, "mcp_")
}

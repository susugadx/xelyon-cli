package agent

import (
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/investigation"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/toolmeta"
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

var helpToolDisplayOrder = toolmeta.HelpDisplayOrder()

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
	if name == "apply_patch" {
		return "Primary edit tool for precise patch-based file changes"
	}
	if summary, ok := surface.HelpSummary(name); ok {
		return summary
	}
	if summary, ok := toolmeta.HelpSummary(name); ok {
		return summary
	}
	return fallbackHelpDescription(description)
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
	return mcpnames.IsExportedToolName(def.Name)
}

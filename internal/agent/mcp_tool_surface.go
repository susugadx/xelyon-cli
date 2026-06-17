package agent

import (
	"fmt"
	"io"
)

func emitMCPToolSurfaceBudgetWarning(selection mcpToolSurfaceSelection, errOut io.Writer) {
	if errOut == nil || !selection.hasOmissions() {
		return
	}
	_, _ = fmt.Fprintf(
		errOut,
		"Warning: MCP tool surface budget exposed %d/%d tools; omitted %d. Use ~/.xelyon/mcp.json tools.include/tools.exclude to narrow MCP tools.\n",
		len(selection.selected),
		selection.total,
		len(selection.omitted),
	)
	for _, line := range selection.warningLines(5) {
		_, _ = fmt.Fprintln(errOut, line)
	}
}

func (s mcpToolSurfaceSelection) warningLines(limit int) []string {
	if limit <= 0 || len(s.omitted) == 0 {
		return nil
	}
	lines := make([]string, 0, limit+1)
	for i, omission := range s.omitted {
		if i >= limit {
			lines = append(lines, fmt.Sprintf("  ... %d more omitted MCP tools", len(s.omitted)-limit))
			break
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s)", omission.exportedName, omission.reason))
	}
	return lines
}

func (a *Agent) refreshMCPToolSurface() {
	if a == nil || a.mcpManager == nil {
		return
	}
	a.mcpSurface = selectMCPToolSurface(a.CurrentModel, a.mcpManager.GetTools())
}

func (a *Agent) currentMCPToolSurface() mcpToolSurfaceSelection {
	if a == nil || a.mcpManager == nil {
		return mcpToolSurfaceSelection{}
	}
	if a.mcpSurface.total == len(a.mcpManager.GetTools()) {
		return a.mcpSurface
	}
	return selectMCPToolSurface(a.CurrentModel, a.mcpManager.GetTools())
}

func (a *Agent) currentMCPBudgetExcludedToolNames() []string {
	return a.currentMCPToolSurface().omittedExportedNames()
}

func (a *Agent) excludedToolsForVisibilityPolicy(policy toolVisibilityPolicy) []string {
	excluded := policy.excluded()
	return appendUniqueStrings(excluded, a.currentMCPBudgetExcludedToolNames()...)
}

func (a *Agent) configureCurrentProviderMCPTools() {
	if a == nil || a.mcpManager == nil {
		return
	}
	surface := a.currentMCPToolSurface()
	configureMCPTools(a.CurrentProvider, surface.selectedTools(), a.errorOutput())
	emitMCPToolSurfaceBudgetWarning(surface, a.errorOutput())
}

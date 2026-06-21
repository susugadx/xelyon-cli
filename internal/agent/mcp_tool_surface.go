package agent

import (
	"fmt"
	"io"
	"reflect"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
)

func emitMCPToolSurfaceBudgetWarning(selection mcpToolSurfaceSelection, errOut io.Writer) {
	if errOut == nil || !selection.hasOmissions() {
		return
	}
	_, _ = fmt.Fprintf(
		errOut,
		"Warning: MCP tool surface budget exposed %d/%d tools; omitted %d. Run /mcp status for details. First narrow ~/.xelyon/mcp.json mcpServers.<server>.tools.include/exclude; if the server is intentionally large, raise ~/.xelyon/config.yaml mcp.surface_budget.\n",
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

func (a *Agent) mcpToolSurfaceBudget() mcpToolSurfaceBudget {
	if a == nil {
		return defaultMCPToolSurfaceBudget()
	}
	return config.EffectiveMCPSurfaceBudget(a.cfg())
}

func (a *Agent) refreshMCPToolSurface() (mcpToolSurfaceSelection, bool) {
	if a == nil {
		return mcpToolSurfaceSelection{}, false
	}
	previous := a.mcpSurface
	if a.mcpManager == nil {
		return previous, false
	}
	next := selectMCPToolSurfaceWithBudget(a.CurrentModel, a.mcpManager.GetTools(), a.mcpToolSurfaceBudget())
	a.mcpSurface = next
	return previous, !sameMCPToolSurfaceSelection(previous, next)
}

func sameMCPToolSurfaceSelection(a, b mcpToolSurfaceSelection) bool {
	return a.total == b.total &&
		a.estimatedTokens == b.estimatedTokens &&
		a.budget == b.budget &&
		a.model == b.model &&
		a.toolSignature == b.toolSignature &&
		reflect.DeepEqual(a.selectedMetrics, b.selectedMetrics) &&
		reflect.DeepEqual(a.omitted, b.omitted)
}

func (a *Agent) currentMCPToolSurface() mcpToolSurfaceSelection {
	if a == nil || a.mcpManager == nil {
		return mcpToolSurfaceSelection{}
	}
	budget := a.mcpToolSurfaceBudget()
	tools := visibleMCPTools(a.mcpManager.GetTools())
	toolSignature := mcpVisibleToolSurfaceSignature(tools)
	if a.mcpSurface.total == len(tools) &&
		a.mcpSurface.budget == mcpsurface.NormalizeBudget(budget) &&
		a.mcpSurface.model == a.CurrentModel &&
		a.mcpSurface.toolSignature == toolSignature {
		return a.mcpSurface
	}
	return selectMCPToolSurfaceWithBudget(a.CurrentModel, tools, budget)
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

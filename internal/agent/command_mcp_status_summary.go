package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"os"
)

func buildMCPStatusSummaryTable(agent *Agent, snapshot mcp.StatusSnapshot, surface mcpToolSurfaceSelection, analysis mcpsurface.Report) *ui.Table {
	budget := surface.budget
	if budget == (mcpsurface.Budget{}) {
		var cfg *config.Config
		if agent != nil {
			cfg = agent.cfg()
		}
		budget = config.EffectiveMCPSurfaceBudget(cfg)
	}
	return ui.NewTable().
		AddRow("Runtime", mcpRuntimeStatusText(agent)).
		AddRow("Config", mcpStatusConfigText(snapshot)).
		AddRow("Servers", fmt.Sprintf(
			"%d configured, %d connected, %d disabled, %d blocked, %d not connected",
			snapshot.ServerCount,
			snapshot.ConnectedServerCount,
			snapshot.DisabledServerCount,
			snapshot.BlockedServerCount,
			snapshot.NotConnectedServerCount,
		)).
		AddRow("Tools", fmt.Sprintf(
			"%d visible / %d registered, %d omitted",
			len(surface.selected),
			snapshot.RegisteredToolCount,
			len(surface.omitted),
		)).
		AddRow("Budget", fmt.Sprintf(
			"%d estimated tokens (%s)",
			surface.estimatedTokens,
			mcpsurface.FormatBudget(budget),
		)).
		AddRow("Surface", fmt.Sprintf(
			"%s schema across %d server(s)",
			mcpsurface.FormatBytes(analysis.SchemaBytes),
			len(analysis.Servers),
		))
}

func mcpRuntimeStatusText(agent *Agent) string {
	if agent == nil {
		return "unknown"
	}
	cfg := agent.cfg()
	if cfg == nil {
		return "unknown"
	}
	if !cfg.MCP.Enabled {
		return "disabled (mcp.enabled=false)"
	}
	if agent.Headless && !cfg.MCP.Headless {
		return "disabled in headless (mcp.headless=false)"
	}
	if os.Getenv("XELYON_DISABLE_MCP") == "1" {
		return "disabled (XELYON_DISABLE_MCP=1)"
	}
	return "enabled"
}

func mcpStatusConfigText(snapshot mcp.StatusSnapshot) string {
	if snapshot.ConfigLoaded {
		return "loaded"
	}
	return "not loaded"
}

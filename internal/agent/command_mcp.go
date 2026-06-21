package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"io"
)

func handleMCPCommand(agent *Agent, args []string) bool {
	if len(args) == 0 || (len(args) == 1 && args[0] == "status") {
		printMCPStatus(agent)
		return true
	}
	printMCPUsage(agent.output())
	return true
}

func printMCPUsage(out io.Writer) {
	yellow.Fprintln(out, "Usage: /mcp status")
}

func printMCPStatus(agent *Agent) {
	out := agent.output()
	printCommandHeaderToWriter(out, "MCP Status / MCP状態")
	_, _ = fmt.Fprintln(out)

	snapshot := mcpStatusSnapshot(agent)
	surface := agent.currentMCPToolSurface()
	analysis := surface.analysis()
	_, _ = fmt.Fprint(out, buildMCPStatusSummaryTable(agent, snapshot, surface, analysis).RenderCompact())
	printMCPStatusServerTable(out, snapshot, surface, analysis)
	printMCPStatusToolSamples(out, surface)
	printMCPStatusToolSurfaceAnalysis(out, analysis)
	_, _ = fmt.Fprintln(out)
}

func mcpStatusSnapshot(agent *Agent) mcp.StatusSnapshot {
	if agent == nil || agent.mcpManager == nil {
		return mcp.StatusSnapshot{}
	}
	return agent.mcpManager.StatusSnapshot()
}

func mcpStatusInlineSummary(agent *Agent) string {
	snapshot := mcpStatusSnapshot(agent)
	surface := agent.currentMCPToolSurface()
	return fmt.Sprintf(
		"%d/%d servers connected, %d/%d tools visible, %d omitted (/mcp status for details)",
		snapshot.ConnectedServerCount,
		snapshot.ServerCount,
		len(surface.selected),
		snapshot.RegisteredToolCount,
		len(surface.omitted),
	)
}

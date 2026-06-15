package agent

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const mcpStatusSampleLimit = 10

type mcpStatusToolSample struct {
	exportedName string
	approval     string
	reason       string
}

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
	_, _ = fmt.Fprint(out, buildMCPStatusSummaryTable(agent, snapshot, surface).RenderCompact())
	printMCPStatusServerTable(out, snapshot, surface)
	printMCPStatusToolSamples(out, surface)
	_, _ = fmt.Fprintln(out)
}

func mcpStatusSnapshot(agent *Agent) mcp.StatusSnapshot {
	if agent == nil || agent.mcpManager == nil {
		return mcp.StatusSnapshot{}
	}
	return agent.mcpManager.StatusSnapshot()
}

func buildMCPStatusSummaryTable(agent *Agent, snapshot mcp.StatusSnapshot, surface mcpToolSurfaceSelection) *ui.Table {
	budget := defaultMCPToolSurfaceBudget()
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
			"%d estimated tokens (max %d tools / %d tokens / %s schema)",
			surface.estimatedTokens,
			budget.maxTools,
			budget.maxEstimatedTokens,
			formatMCPStatusBytes(budget.maxSchemaBytes),
		))
}

func printMCPStatusServerTable(out io.Writer, snapshot mcp.StatusSnapshot, surface mcpToolSurfaceSelection) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🔌 MCP servers")
	if len(snapshot.Servers) == 0 {
		dim.Fprintln(out, "  No MCP servers are loaded in this session")
		return
	}

	visibleByServer, omittedByServer := mcpStatusSurfaceCounts(surface)
	table := ui.NewTable().SetHeaders("Server", "State", "Tools", "Approval", "Timeouts", "Last healthy")
	for _, server := range snapshot.Servers {
		table.AddRow(
			server.Name,
			string(server.State),
			mcpStatusServerToolText(server, visibleByServer[server.Name], omittedByServer[server.Name]),
			mcpStatusApprovalText(server),
			fmt.Sprintf("startup %ds / tool %ds", server.StartupTimeoutSeconds, server.ToolTimeoutSeconds),
			mcpStatusLastHealthyText(server),
		)
	}
	_, _ = fmt.Fprint(out, table.RenderCompact())
}

func printMCPStatusToolSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🧰 MCP tools")
	printMCPStatusVisibleSamples(out, surface)
	printMCPStatusOmittedSamples(out, surface)
}

func printMCPStatusVisibleSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	samples := mcpStatusVisibleSamples(surface.selected, mcpStatusSampleLimit)
	if len(samples) == 0 {
		dim.Fprintln(out, "  Visible: none")
		return
	}
	_, _ = fmt.Fprintf(out, "  Visible: %d\n", len(surface.selected))
	for _, sample := range samples {
		_, _ = fmt.Fprintf(out, "    - %s (%s)\n", sample.exportedName, sample.approval)
	}
	if remaining := len(surface.selected) - len(samples); remaining > 0 {
		_, _ = fmt.Fprintf(out, "    ... %d more visible MCP tools\n", remaining)
	}
}

func printMCPStatusOmittedSamples(out io.Writer, surface mcpToolSurfaceSelection) {
	samples := mcpStatusOmittedSamples(surface.omitted, mcpStatusSampleLimit)
	if len(samples) == 0 {
		dim.Fprintln(out, "  Omitted: none")
		return
	}
	_, _ = fmt.Fprintf(out, "  Omitted: %d\n", len(surface.omitted))
	for _, sample := range samples {
		_, _ = fmt.Fprintf(out, "    - %s (%s)\n", sample.exportedName, sample.reason)
	}
	if remaining := len(surface.omitted) - len(samples); remaining > 0 {
		_, _ = fmt.Fprintf(out, "    ... %d more omitted MCP tools\n", remaining)
	}
}

func mcpStatusVisibleSamples(tools []mcp.MCPTool, limit int) []mcpStatusToolSample {
	if limit <= 0 || len(tools) == 0 {
		return nil
	}
	samples := make([]mcpStatusToolSample, 0, len(tools))
	for _, tool := range tools {
		samples = append(samples, mcpStatusToolSample{
			exportedName: mcpnames.ExportedToolName(tool.ServerName, tool.Name),
			approval:     tool.ApprovalMode().String(),
		})
	}
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].exportedName < samples[j].exportedName
	})
	if len(samples) > limit {
		return samples[:limit]
	}
	return samples
}

func mcpStatusOmittedSamples(omissions []mcpToolSurfaceOmission, limit int) []mcpStatusToolSample {
	if limit <= 0 || len(omissions) == 0 {
		return nil
	}
	samples := make([]mcpStatusToolSample, 0, len(omissions))
	for _, omission := range omissions {
		if strings.TrimSpace(omission.exportedName) == "" {
			continue
		}
		samples = append(samples, mcpStatusToolSample{
			exportedName: omission.exportedName,
			reason:       omission.reason,
		})
	}
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].exportedName < samples[j].exportedName
	})
	if len(samples) > limit {
		return samples[:limit]
	}
	return samples
}

func mcpStatusSurfaceCounts(surface mcpToolSurfaceSelection) (map[string]int, map[string]int) {
	visible := make(map[string]int)
	for _, tool := range surface.selected {
		visible[tool.ServerName]++
	}
	omitted := make(map[string]int)
	for _, omission := range surface.omitted {
		if strings.TrimSpace(omission.serverName) == "" {
			continue
		}
		omitted[omission.serverName]++
	}
	return visible, omitted
}

func mcpStatusServerToolText(server mcp.ServerStatusSnapshot, visible, omitted int) string {
	text := fmt.Sprintf("%d visible / %d registered", visible, server.RegisteredToolCount)
	if omitted > 0 {
		text += fmt.Sprintf(", %d omitted", omitted)
	}
	return text
}

func mcpStatusApprovalText(server mcp.ServerStatusSnapshot) string {
	if server.ApprovalValid {
		return server.Approval
	}
	return server.Approval + " (invalid config)"
}

func mcpStatusLastHealthyText(server mcp.ServerStatusSnapshot) string {
	if !server.LastHealthySet {
		return "never"
	}
	elapsed := time.Since(server.LastHealthy).Round(time.Second)
	if elapsed <= 0 {
		return "now"
	}
	return elapsed.String() + " ago"
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

func formatMCPStatusBytes(bytes int) string {
	if bytes >= 1024 && bytes%1024 == 0 {
		return fmt.Sprintf("%d KiB", bytes/1024)
	}
	return fmt.Sprintf("%d bytes", bytes)
}

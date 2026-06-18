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
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/termtext"
)

const mcpStatusSampleLimit = 10
const mcpStatusSurfaceLimit = 5

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

func buildMCPStatusSummaryTable(agent *Agent, snapshot mcp.StatusSnapshot, surface mcpToolSurfaceSelection, analysis mcpsurface.Report) *termtext.Table {
	budget := defaultMCPToolSurfaceBudget()
	return termtext.NewTable().
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
			mcpsurface.FormatBytes(budget.maxSchemaBytes),
		)).
		AddRow("Surface", fmt.Sprintf(
			"%s schema across %d server(s)",
			mcpsurface.FormatBytes(analysis.SchemaBytes),
			len(analysis.Servers),
		))
}

func printMCPStatusServerTable(out io.Writer, snapshot mcp.StatusSnapshot, surface mcpToolSurfaceSelection, analysis mcpsurface.Report) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🔌 MCP servers")
	if len(snapshot.Servers) == 0 {
		dim.Fprintln(out, "  No MCP servers are loaded in this session")
		return
	}

	visibleByServer, omittedByServer := mcpStatusSurfaceCounts(surface)
	surfaceByServer := mcpStatusServerSurfaceByName(analysis)
	table := termtext.NewTable().SetHeaders("Server", "State", "Tools", "Tokens", "Schema", "Omitted reasons", "Approval", "Timeouts", "Last healthy")
	for _, server := range snapshot.Servers {
		serverSurface := surfaceByServer[server.Name]
		table.AddRow(
			server.Name,
			string(server.State),
			mcpStatusServerToolText(server, visibleByServer[server.Name], omittedByServer[server.Name]),
			mcpStatusServerTokensText(serverSurface),
			mcpStatusServerSchemaText(serverSurface),
			mcpsurface.FormatReasonCounts(serverSurface.OmittedReasons, mcpStatusSurfaceLimit),
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

func printMCPStatusToolSurfaceAnalysis(out io.Writer, analysis mcpsurface.Report) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📊 MCP tool surface")
	if analysis.TotalTools == 0 {
		dim.Fprintln(out, "  No MCP tool surface is visible in this session")
		return
	}
	_, _ = fmt.Fprintf(
		out,
		"  Summary: %d visible / %d registered / %d total, %d omitted, %d estimated tokens (all analyzed), %s schema\n",
		analysis.VisibleTools,
		analysis.RegisteredTools,
		analysis.TotalTools,
		analysis.OmittedTools,
		analysis.EstimatedTokens,
		mcpsurface.FormatBytes(analysis.SchemaBytes),
	)
	printMCPStatusTopServers(out, analysis)
	_, _ = fmt.Fprintf(out, "  Top omitted reasons: %s\n", mcpsurface.FormatReasonCounts(analysis.OmittedReasons, mcpStatusSurfaceLimit))
	printMCPStatusToolMetrics(out, "Largest schema tools", analysis.LargestSchemaTools, true)
	printMCPStatusToolMetrics(out, "Highest estimated token tools", analysis.HighestEstimatedTokenTools, false)
	printMCPStatusRecommendations(out, analysis)
}

func printMCPStatusTopServers(out io.Writer, analysis mcpsurface.Report) {
	servers := append([]mcpsurface.ServerSummary(nil), analysis.Servers...)
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].EstimatedTokens != servers[j].EstimatedTokens {
			return servers[i].EstimatedTokens > servers[j].EstimatedTokens
		}
		if servers[i].SchemaBytes != servers[j].SchemaBytes {
			return servers[i].SchemaBytes > servers[j].SchemaBytes
		}
		if servers[i].RegisteredTools != servers[j].RegisteredTools {
			return servers[i].RegisteredTools > servers[j].RegisteredTools
		}
		return servers[i].ServerName < servers[j].ServerName
	})
	if len(servers) > mcpStatusSurfaceLimit {
		servers = servers[:mcpStatusSurfaceLimit]
	}
	_, _ = fmt.Fprintln(out, "  Top heavy servers:")
	for _, server := range servers {
		_, _ = fmt.Fprintf(
			out,
			"    - %s: %d tokens, %s schema, %d visible / %d omitted\n",
			server.ServerName,
			server.EstimatedTokens,
			mcpsurface.FormatBytes(server.SchemaBytes),
			server.VisibleTools,
			server.OmittedTools,
		)
	}
}

func printMCPStatusToolMetrics(out io.Writer, label string, metrics []mcpsurface.ToolMetric, schema bool) {
	_, _ = fmt.Fprintf(out, "  %s:\n", label)
	if len(metrics) == 0 {
		dim.Fprintln(out, "    none")
		return
	}
	for _, metric := range metrics {
		name := mcpsurface.MetricName(metric)
		if schema {
			_, _ = fmt.Fprintf(out, "    - %s: %s schema\n", name, mcpsurface.FormatBytes(metric.SchemaBytes))
			continue
		}
		_, _ = fmt.Fprintf(out, "    - %s: %d tokens\n", name, metric.EstimatedTokens)
	}
}

func printMCPStatusRecommendations(out io.Writer, analysis mcpsurface.Report) {
	_, _ = fmt.Fprintln(out, "  Recommendations:")
	if len(analysis.Recommendations) == 0 {
		dim.Fprintln(out, "    none")
		return
	}
	for _, recommendation := range analysis.Recommendations {
		_, _ = fmt.Fprintf(out, "    - %s: %s\n", recommendation.ServerName, recommendation.Reason)
		_, _ = fmt.Fprintf(out, "      ~/.xelyon/mcp.json mcpServers fragment: %s\n", mcpsurface.IncludeSnippet(recommendation))
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

func mcpStatusServerSurfaceByName(analysis mcpsurface.Report) map[string]mcpsurface.ServerSummary {
	byName := make(map[string]mcpsurface.ServerSummary, len(analysis.Servers))
	for _, server := range analysis.Servers {
		byName[server.ServerName] = server
	}
	return byName
}

func mcpStatusServerToolText(server mcp.ServerStatusSnapshot, visible, omitted int) string {
	text := fmt.Sprintf("%d visible / %d registered", visible, server.RegisteredToolCount)
	if omitted > 0 {
		text += fmt.Sprintf(", %d omitted", omitted)
	}
	return text
}

func mcpStatusServerTokensText(server mcpsurface.ServerSummary) string {
	if server.RegisteredTools == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", server.EstimatedTokens)
}

func mcpStatusServerSchemaText(server mcpsurface.ServerSummary) string {
	if server.RegisteredTools == 0 {
		return "-"
	}
	return mcpsurface.FormatBytes(server.SchemaBytes)
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

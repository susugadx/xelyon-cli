package agent

import (
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/termtext"
	"io"
	"time"
)

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

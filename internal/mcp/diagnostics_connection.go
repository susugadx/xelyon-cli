package mcp

import (
	"context"
	"errors"
	"fmt"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"strings"
)

func diagnoseServerConnection(
	ctx context.Context,
	manager *Manager,
	serverName string,
	serverConfig ServerConfig,
	includeTools bool,
	serverReport *DiagnosticServerReport,
) []mcpsurface.Tool {
	connection := &DiagnosticConnectionReport{Attempted: false, Status: "skipped"}
	serverReport.Connection = connection

	if serverConfig.Disabled {
		connection.Error = "server disabled"
		return nil
	}
	if err := validateMCPCommand(serverConfig.Command); err != nil {
		connection.Error = err.Error()
		return nil
	}

	connection.Attempted = true
	client := manager.newClient()
	session, err := manager.openServerSession(ctx, client, serverConfig)
	if err != nil {
		detail := diagnosticMCPServerErrorDetail("initialize", err)
		connection.Status = "fail"
		connection.Error = detail
		serverReport.addCheck(DiagnosticStatusFail, "connect", "MCP server connection failed", detail, "Check command, args, env, and startupTimeoutSeconds")
		return nil
	}

	listCtx, cancel := mcpServerOperationContext(ctx, serverConfig.startupTimeoutDuration())
	defer cancel()
	toolsResult, err := session.ListTools(listCtx, nil)
	if err != nil {
		_ = session.Close()
		detail := diagnosticMCPServerErrorDetail("tools/list", err)
		connection.Status = "fail"
		connection.Error = detail
		serverReport.addCheck(DiagnosticStatusFail, "tools_list", "MCP server tools/list failed", detail, "Check the MCP server logs and startupTimeoutSeconds")
		return nil
	}
	var listedTools []*sdkmcp.Tool
	if toolsResult != nil {
		listedTools = toolsResult.Tools
	}
	serverTools, summary := manager.buildServerTools(serverName, session, listedTools, serverConfig)

	previous := manager.swapServerSession(serverName, session)
	manager.replaceServerTools(serverName, serverTools)
	if previous != nil && previous != session {
		_ = previous.Close()
	}
	manager.markServerHealthy(serverName)

	connection.Status = "ok"
	connection.RegisteredToolCount = summary.registered
	connection.SkippedToolCount = summary.skipped
	serverReport.addCheck(DiagnosticStatusOK, "connect", "MCP server connected", "", "")
	serverReport.addCheck(DiagnosticStatusOK, "tools_list", "MCP server tools/list succeeded", fmt.Sprintf("registered=%d skipped=%d", summary.registered, summary.skipped), "")

	return populateConnectionDiagnostics(manager, serverName, serverConfig, listedTools, connection, serverReport, includeTools)
}

func diagnosticMCPServerErrorDetail(operation string, err error) string {
	status := operation + " failed"
	if errors.Is(err, context.DeadlineExceeded) {
		status = operation + " timed out"
	} else if errors.Is(err, context.Canceled) {
		status = operation + " canceled"
	}
	return status + "; server error detail suppressed by doctor mcp privacy policy"
}

func populateConnectionDiagnostics(
	manager *Manager,
	serverName string,
	serverConfig ServerConfig,
	listedTools []*sdkmcp.Tool,
	connection *DiagnosticConnectionReport,
	serverReport *DiagnosticServerReport,
	includeTools bool,
) []mcpsurface.Tool {
	rawNames := make(map[string]bool, len(listedTools))
	rawToolCount := 0
	for _, tool := range listedTools {
		if tool != nil {
			rawToolCount++
			rawNames[tool.Name] = true
		}
	}
	connection.RawToolCount = rawToolCount
	connection.UnknownToolApprovals = sortedUnknownKeys(serverConfig.ToolApprovals, rawNames)
	connection.UnknownIncludes = sortedUnknownNames(serverConfig.includeTools(), rawNames)
	if len(serverConfig.includeTools()) == 0 {
		connection.UnknownExcludes = sortedUnknownNames(serverConfig.excludeTools(), rawNames)
	}
	addUnknownToolReferenceChecks(serverReport, connection)

	decisions, _ := manager.planServerToolRegistration(serverName, listedTools, serverConfig)
	toolSurfaceTools := make([]mcpsurface.Tool, 0, len(decisions))
	for _, decision := range decisions {
		switch decision.skipReason {
		case toolSkipFiltered:
			connection.FilteredToolCount++
		case toolSkipServerDeny, toolSkipToolDeny:
			connection.DeniedToolCount++
		case toolSkipCollision:
			connection.CollisionToolCount++
		}
		toolSurfaceTools = append(toolSurfaceTools, diagnosticToolSurfaceTool(serverName, decision))
	}
	if includeTools {
		for _, decision := range decisions {
			serverReport.Tools = append(serverReport.Tools, diagnosticToolReport(decision))
		}
	}
	return toolSurfaceTools
}

func addUnknownToolReferenceChecks(serverReport *DiagnosticServerReport, connection *DiagnosticConnectionReport) {
	if len(connection.UnknownToolApprovals) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_approvals", "toolApprovals contains unknown tool names", strings.Join(connection.UnknownToolApprovals, ", "), "Use raw tool names returned by the MCP server")
	}
	if len(connection.UnknownIncludes) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_filter", "tools.include contains unknown tool names", strings.Join(connection.UnknownIncludes, ", "), "Use raw tool names returned by the MCP server")
	}
	if len(connection.UnknownExcludes) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_filter", "tools.exclude contains unknown tool names", strings.Join(connection.UnknownExcludes, ", "), "Use raw tool names returned by the MCP server")
	}
}

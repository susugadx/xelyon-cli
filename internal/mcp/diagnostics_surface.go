package mcp

import (
	"encoding/json"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/token"
	"strings"
)

func diagnosticBudgetHiddenReasons(selection mcpsurface.BudgetSelection) map[string]string {
	reasons := make(map[string]string)
	for _, tool := range selection.Omitted {
		if !tool.Registered || strings.TrimSpace(tool.ExportedName) == "" {
			continue
		}
		switch tool.OmittedReason {
		case mcpsurface.OmittedReasonSchemaTooLarge, mcpsurface.OmittedReasonToolCountBudgetExceeded, mcpsurface.OmittedReasonTokenBudgetExceeded:
			reasons[tool.ExportedName] = tool.OmittedReason
		}
	}
	return reasons
}

func diagnosticToolReport(decision toolRegistrationDecision) DiagnosticToolReport {
	approval := ""
	if decision.approval != "" {
		approval = string(mcpapproval.Effective(decision.approval))
	}
	report := DiagnosticToolReport{
		Name:         decision.tool.Name,
		ExportedName: decision.exportedName,
		Approval:     approval,
		Visible:      decision.registered(),
	}
	if !decision.registered() {
		report.HiddenReason = string(decision.skipReason)
	}
	return report
}

func diagnosticToolSurfaceTool(serverName string, decision toolRegistrationDecision) mcpsurface.Tool {
	exportedName := diagnosticDecisionExportedName(serverName, decision)
	inputSchema := diagnosticToolInputSchemaBytes(decision.tool)
	visible := decision.registered()
	reason := ""
	if !visible {
		reason = string(decision.skipReason)
	}
	toolName := ""
	if decision.tool != nil {
		toolName = decision.tool.Name
	}
	return mcpsurface.Tool{
		ServerName:      serverName,
		ToolName:        toolName,
		ExportedName:    exportedName,
		Registered:      decision.registered(),
		Visible:         visible,
		OmittedReason:   reason,
		SchemaBytes:     len(inputSchema),
		EstimatedTokens: diagnosticToolEstimatedTokens(exportedName, decision.tool, inputSchema),
	}
}

func applyDiagnosticRuntimeToolSurface(report *DiagnosticReport, tools []MCPTool, budget mcpsurface.Budget, includeTools bool, connectionSurfaceTools map[string][]mcpsurface.Tool) {
	if report == nil {
		return
	}
	effectiveBudget := mcpsurface.NormalizeBudget(budget)
	hiddenReasons := map[string]string{}
	if len(tools) > 0 {
		surfaceTools := make([]mcpsurface.Tool, 0, len(tools))
		for _, tool := range tools {
			surfaceTools = append(surfaceTools, diagnosticRuntimeToolSurfaceTool(tool))
		}
		budgeted := mcpsurface.ApplyBudget(surfaceTools, effectiveBudget)
		surface := mcpsurface.Analyze(budgeted.AnalysisTools(), mcpsurface.Options{Budget: budgeted.Budget})
		report.ToolSurface = &surface
		hiddenReasons = diagnosticBudgetHiddenReasons(budgeted)
	}
	applyDiagnosticConnectionToolSurfaces(report, connectionSurfaceTools, hiddenReasons, effectiveBudget)
	if !includeTools {
		return
	}
	for serverIndex := range report.Servers {
		for toolIndex := range report.Servers[serverIndex].Tools {
			tool := &report.Servers[serverIndex].Tools[toolIndex]
			if reason := hiddenReasons[tool.ExportedName]; tool.Visible && reason != "" {
				tool.Visible = false
				tool.HiddenReason = reason
			}
		}
	}
}

func applyDiagnosticConnectionToolSurfaces(report *DiagnosticReport, connectionSurfaceTools map[string][]mcpsurface.Tool, hiddenReasons map[string]string, budget mcpsurface.Budget) {
	if report == nil || len(connectionSurfaceTools) == 0 {
		return
	}
	for serverIndex := range report.Servers {
		connection := report.Servers[serverIndex].Connection
		if connection == nil {
			continue
		}
		tools := connectionSurfaceTools[report.Servers[serverIndex].Name]
		if len(tools) == 0 {
			continue
		}
		projected := projectDiagnosticSurfaceTools(tools, hiddenReasons)
		surface := mcpsurface.Analyze(projected, mcpsurface.Options{Budget: budget})
		connection.ToolSurface = &surface
	}
}

func projectDiagnosticSurfaceTools(tools []mcpsurface.Tool, hiddenReasons map[string]string) []mcpsurface.Tool {
	projected := make([]mcpsurface.Tool, 0, len(tools))
	for _, tool := range tools {
		if reason := hiddenReasons[tool.ExportedName]; tool.Registered && tool.Visible && reason != "" {
			tool.Visible = false
			tool.OmittedReason = reason
		}
		projected = append(projected, tool)
	}
	return projected
}

func diagnosticRuntimeToolSurfaceTool(tool MCPTool) mcpsurface.Tool {
	exportedName := mcpnames.ExportedToolName(tool.ServerName, tool.Name)
	return mcpsurface.Tool{
		ServerName:      tool.ServerName,
		ToolName:        tool.Name,
		ExportedName:    exportedName,
		Registered:      true,
		Visible:         true,
		SchemaBytes:     len(tool.InputSchema),
		EstimatedTokens: diagnosticMCPToolEstimatedTokens(exportedName, tool),
	}
}

func diagnosticDecisionExportedName(serverName string, decision toolRegistrationDecision) string {
	if strings.TrimSpace(decision.exportedName) != "" {
		return decision.exportedName
	}
	if decision.tool == nil {
		return ""
	}
	return mcpnames.ExportedToolName(serverName, decision.tool.Name)
}

func diagnosticToolInputSchemaBytes(tool *sdkmcp.Tool) []byte {
	if tool == nil || tool.InputSchema == nil {
		return nil
	}
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil
	}
	return schemaBytes
}

func diagnosticToolEstimatedTokens(exportedName string, tool *sdkmcp.Tool, inputSchema []byte) int {
	if tool == nil {
		return 0
	}
	definition := api.ConvertMCPToolToToolDefinition(exportedName, tool.Description, inputSchema)
	return token.EstimateStructuredValueTokenCount(definition)
}

func diagnosticMCPToolEstimatedTokens(exportedName string, tool MCPTool) int {
	definition := api.ConvertMCPToolToToolDefinition(exportedName, tool.Description, tool.InputSchema)
	return token.EstimateStructuredValueTokenCount(definition)
}

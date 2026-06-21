package agent

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/token"
)

type mcpToolSurfaceSelection struct {
	selected        []mcp.MCPTool
	selectedMetrics []mcpToolSurfaceMetric
	omitted         []mcpToolSurfaceOmission
	total           int
	estimatedTokens int
	budget          mcpsurface.Budget
	model           string
	toolSignature   string
}

type mcpToolSurfaceMetric struct {
	exportedName    string
	serverName      string
	toolName        string
	schemaBytes     int
	estimatedTokens int
}

type mcpToolSurfaceOmission struct {
	exportedName    string
	serverName      string
	toolName        string
	reason          string
	schemaBytes     int
	estimatedTokens int
}

func defaultMCPToolSurfaceBudget() mcpToolSurfaceBudget {
	return mcpsurface.DefaultBudget()
}

type mcpToolSurfaceBudget = mcpsurface.Budget

func selectMCPToolSurfaceWithBudget(model string, tools []mcp.MCPTool, budget mcpToolSurfaceBudget) mcpToolSurfaceSelection {
	tools = visibleMCPTools(tools)
	toolSignature := mcpVisibleToolSurfaceSignature(tools)
	budget = mcpsurface.NormalizeBudget(budget)
	surfaceTools := make([]mcpsurface.Tool, 0, len(tools))
	toolByExportedName := make(map[string]mcp.MCPTool, len(tools))
	for _, tool := range tools {
		exportedName := mcpnames.ExportedToolName(tool.ServerName, tool.Name)
		schemaBytes := len(tool.InputSchema)
		if schemaBytes > budget.MaxSchemaBytesPerTool {
			surfaceTools = append(surfaceTools, mcpsurface.Tool{
				ServerName:   tool.ServerName,
				ToolName:     tool.Name,
				ExportedName: exportedName,
				Registered:   true,
				Visible:      true,
				SchemaBytes:  schemaBytes,
			})
			if _, ok := toolByExportedName[exportedName]; !ok {
				toolByExportedName[exportedName] = tool
			}
			continue
		}
		def := api.ConvertMCPToolToToolDefinition(exportedName, tool.Description, tool.InputSchema)
		estimatedTokens := token.EstimateStructuredValueTokenCountForModel(model, def)
		surfaceTools = append(surfaceTools, mcpsurface.Tool{
			ServerName:      tool.ServerName,
			ToolName:        tool.Name,
			ExportedName:    exportedName,
			Registered:      true,
			Visible:         true,
			SchemaBytes:     schemaBytes,
			EstimatedTokens: estimatedTokens,
		})
		if _, ok := toolByExportedName[exportedName]; !ok {
			toolByExportedName[exportedName] = tool
		}
	}

	budgeted := mcpsurface.ApplyBudget(surfaceTools, budget)
	selection := mcpToolSurfaceSelection{
		total:           len(tools),
		estimatedTokens: budgeted.EstimatedTokens,
		budget:          budgeted.Budget,
		model:           model,
		toolSignature:   toolSignature,
	}
	for _, selected := range budgeted.Selected {
		tool, ok := toolByExportedName[selected.ExportedName]
		if !ok {
			continue
		}
		selection.selected = append(selection.selected, tool)
		selection.selectedMetrics = append(selection.selectedMetrics, mcpToolSurfaceMetric{
			exportedName:    selected.ExportedName,
			serverName:      selected.ServerName,
			toolName:        selected.ToolName,
			schemaBytes:     selected.SchemaBytes,
			estimatedTokens: selected.EstimatedTokens,
		})
	}
	for _, omitted := range budgeted.Omitted {
		selection.omitted = append(selection.omitted, mcpToolSurfaceOmission{
			exportedName:    omitted.ExportedName,
			serverName:      omitted.ServerName,
			toolName:        omitted.ToolName,
			reason:          omitted.OmittedReason,
			schemaBytes:     omitted.SchemaBytes,
			estimatedTokens: omitted.EstimatedTokens,
		})
	}
	return selection
}

func mcpVisibleToolSurfaceSignature(tools []mcp.MCPTool) string {
	hash := sha256.New()
	var scratch [8]byte
	writeInt64 := func(value int64) {
		binary.LittleEndian.PutUint64(scratch[:], uint64(value))
		_, _ = hash.Write(scratch[:])
	}
	writeString := func(value string) {
		binary.LittleEndian.PutUint64(scratch[:], uint64(len(value)))
		_, _ = hash.Write(scratch[:])
		_, _ = hash.Write([]byte(value))
	}
	writeBytes := func(value []byte) {
		binary.LittleEndian.PutUint64(scratch[:], uint64(len(value)))
		_, _ = hash.Write(scratch[:])
		_, _ = hash.Write(value)
	}

	writeInt64(int64(len(tools)))
	for _, tool := range tools {
		writeString(tool.ServerName)
		writeString(tool.Name)
		writeString(tool.Description)
		writeBytes(tool.InputSchema)
		writeInt64(int64(tool.CallTimeout))
		writeString(tool.ApprovalMode().String())
	}
	return string(hash.Sum(nil))
}

func (s mcpToolSurfaceSelection) selectedTools() []mcp.MCPTool {
	return append([]mcp.MCPTool(nil), s.selected...)
}

func (s mcpToolSurfaceSelection) omittedExportedNames() []string {
	names := make([]string, 0, len(s.omitted))
	for _, omission := range s.omitted {
		if strings.TrimSpace(omission.exportedName) == "" {
			continue
		}
		names = append(names, omission.exportedName)
	}
	sort.Strings(names)
	return names
}

func (s mcpToolSurfaceSelection) hasOmissions() bool {
	return len(s.omitted) > 0
}

func (s mcpToolSurfaceSelection) analysis() mcpsurface.Report {
	return mcpsurface.Analyze(s.analysisTools(), mcpsurface.Options{Budget: s.budget})
}

func (s mcpToolSurfaceSelection) analysisTools() []mcpsurface.Tool {
	tools := make([]mcpsurface.Tool, 0, len(s.selected)+len(s.omitted))
	selectedMetrics := s.selectedMetrics
	if len(selectedMetrics) == 0 && len(s.selected) > 0 {
		selectedMetrics = make([]mcpToolSurfaceMetric, 0, len(s.selected))
		for _, tool := range s.selected {
			selectedMetrics = append(selectedMetrics, mcpToolSurfaceMetric{
				exportedName: mcpnames.ExportedToolName(tool.ServerName, tool.Name),
				serverName:   tool.ServerName,
				toolName:     tool.Name,
				schemaBytes:  len(tool.InputSchema),
			})
		}
	}
	for _, metric := range selectedMetrics {
		tools = append(tools, mcpsurface.Tool{
			ServerName:      metric.serverName,
			ToolName:        metric.toolName,
			ExportedName:    metric.exportedName,
			Registered:      true,
			Visible:         true,
			SchemaBytes:     metric.schemaBytes,
			EstimatedTokens: metric.estimatedTokens,
		})
	}
	for _, omission := range s.omitted {
		tools = append(tools, mcpsurface.Tool{
			ServerName:      omission.serverName,
			ToolName:        omission.toolName,
			ExportedName:    omission.exportedName,
			Registered:      true,
			Visible:         false,
			OmittedReason:   omission.reason,
			SchemaBytes:     omission.schemaBytes,
			EstimatedTokens: omission.estimatedTokens,
		})
	}
	return tools
}

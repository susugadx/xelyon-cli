package agent

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	defaultMCPToolSurfaceMaxTools           = 80
	defaultMCPToolSurfaceMaxEstimatedTokens = 32000
	defaultMCPToolSurfaceMaxSchemaBytes     = 128 * 1024
)

type mcpToolSurfaceBudget struct {
	maxTools           int
	maxEstimatedTokens int
	maxSchemaBytes     int
}

type mcpToolSurfaceSelection struct {
	selected        []mcp.MCPTool
	omitted         []mcpToolSurfaceOmission
	total           int
	estimatedTokens int
}

type mcpToolSurfaceOmission struct {
	exportedName      string
	serverName        string
	toolName          string
	reason            string
	schemaBytes       int
	estimatedTokens   int
	projectedTokens   int
	selectedToolCount int
}

func defaultMCPToolSurfaceBudget() mcpToolSurfaceBudget {
	return mcpToolSurfaceBudget{
		maxTools:           defaultMCPToolSurfaceMaxTools,
		maxEstimatedTokens: defaultMCPToolSurfaceMaxEstimatedTokens,
		maxSchemaBytes:     defaultMCPToolSurfaceMaxSchemaBytes,
	}
}

func selectMCPToolSurface(model string, tools []mcp.MCPTool) mcpToolSurfaceSelection {
	return selectMCPToolSurfaceWithBudget(model, tools, defaultMCPToolSurfaceBudget())
}

func selectMCPToolSurfaceWithBudget(model string, tools []mcp.MCPTool, budget mcpToolSurfaceBudget) mcpToolSurfaceSelection {
	budget = normalizeMCPToolSurfaceBudget(budget)
	ordered := orderMCPToolsRoundRobin(tools)
	selection := mcpToolSurfaceSelection{total: len(tools)}
	seen := make(map[string]struct{}, len(ordered))

	for _, tool := range ordered {
		exportedName := mcpnames.ExportedToolName(tool.ServerName, tool.Name)
		if _, ok := seen[exportedName]; ok {
			continue
		}
		seen[exportedName] = struct{}{}

		schemaBytes := len(tool.InputSchema)
		if schemaBytes > budget.maxSchemaBytes {
			selection.omitted = append(selection.omitted, mcpToolSurfaceOmission{
				exportedName: exportedName,
				serverName:   tool.ServerName,
				toolName:     tool.Name,
				reason:       "schema_too_large",
				schemaBytes:  schemaBytes,
			})
			continue
		}

		def := api.ConvertMCPToolToToolDefinition(exportedName, tool.Description, tool.InputSchema)
		estimatedTokens := token.EstimateStructuredValueTokenCountForModel(model, def)
		projectedTokens := selection.estimatedTokens + estimatedTokens
		if len(selection.selected) >= budget.maxTools {
			selection.omitted = append(selection.omitted, mcpToolSurfaceOmission{
				exportedName:      exportedName,
				serverName:        tool.ServerName,
				toolName:          tool.Name,
				reason:            "tool_count_budget_exceeded",
				schemaBytes:       schemaBytes,
				estimatedTokens:   estimatedTokens,
				projectedTokens:   projectedTokens,
				selectedToolCount: len(selection.selected),
			})
			continue
		}
		if projectedTokens > budget.maxEstimatedTokens {
			selection.omitted = append(selection.omitted, mcpToolSurfaceOmission{
				exportedName:      exportedName,
				serverName:        tool.ServerName,
				toolName:          tool.Name,
				reason:            "token_budget_exceeded",
				schemaBytes:       schemaBytes,
				estimatedTokens:   estimatedTokens,
				projectedTokens:   projectedTokens,
				selectedToolCount: len(selection.selected),
			})
			continue
		}

		selection.selected = append(selection.selected, tool)
		selection.estimatedTokens = projectedTokens
	}

	sort.SliceStable(selection.omitted, func(i, j int) bool {
		if selection.omitted[i].serverName != selection.omitted[j].serverName {
			return selection.omitted[i].serverName < selection.omitted[j].serverName
		}
		if selection.omitted[i].toolName != selection.omitted[j].toolName {
			return selection.omitted[i].toolName < selection.omitted[j].toolName
		}
		return selection.omitted[i].exportedName < selection.omitted[j].exportedName
	})
	return selection
}

func normalizeMCPToolSurfaceBudget(budget mcpToolSurfaceBudget) mcpToolSurfaceBudget {
	if budget.maxTools <= 0 {
		budget.maxTools = defaultMCPToolSurfaceMaxTools
	}
	if budget.maxEstimatedTokens <= 0 {
		budget.maxEstimatedTokens = defaultMCPToolSurfaceMaxEstimatedTokens
	}
	if budget.maxSchemaBytes <= 0 {
		budget.maxSchemaBytes = defaultMCPToolSurfaceMaxSchemaBytes
	}
	return budget
}

func orderMCPToolsRoundRobin(tools []mcp.MCPTool) []mcp.MCPTool {
	if len(tools) == 0 {
		return nil
	}

	byServer := make(map[string][]mcp.MCPTool)
	for _, tool := range tools {
		byServer[tool.ServerName] = append(byServer[tool.ServerName], tool)
	}

	servers := make([]string, 0, len(byServer))
	for server := range byServer {
		servers = append(servers, server)
		sort.SliceStable(byServer[server], func(i, j int) bool {
			if byServer[server][i].Name != byServer[server][j].Name {
				return byServer[server][i].Name < byServer[server][j].Name
			}
			return byServer[server][i].Description < byServer[server][j].Description
		})
	}
	sort.Strings(servers)

	ordered := make([]mcp.MCPTool, 0, len(tools))
	for index := 0; len(ordered) < len(tools); index++ {
		added := false
		for _, server := range servers {
			serverTools := byServer[server]
			if index >= len(serverTools) {
				continue
			}
			ordered = append(ordered, serverTools[index])
			added = true
		}
		if !added {
			break
		}
	}
	return ordered
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

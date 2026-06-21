package mcpsurface

import (
	"sort"
	"strings"
)

// DefaultBudget は MCP tool surface の既定 budget を返す。
func DefaultBudget() Budget {
	return Budget{
		MaxTools:              defaultBudgetMaxTools,
		EstimatedTokens:       defaultBudgetEstimatedTokens,
		MaxSchemaBytesPerTool: defaultBudgetMaxSchemaBytesPerTool,
	}
}

// NormalizeBudget は 0 以下の budget 値を既定値へ解決する。
func NormalizeBudget(budget Budget) Budget {
	defaults := DefaultBudget()
	if budget.MaxTools <= 0 {
		budget.MaxTools = defaults.MaxTools
	}
	if budget.EstimatedTokens <= 0 {
		budget.EstimatedTokens = defaults.EstimatedTokens
	}
	if budget.MaxSchemaBytesPerTool <= 0 {
		budget.MaxSchemaBytesPerTool = defaults.MaxSchemaBytesPerTool
	}
	return budget
}

func (b Budget) isZero() bool {
	return b.MaxTools == 0 && b.EstimatedTokens == 0 && b.MaxSchemaBytesPerTool == 0
}

// ApplyBudget は sanitized MCP tool metrics に provider-facing budget を適用する。
func ApplyBudget(tools []Tool, budget Budget) BudgetSelection {
	budget = NormalizeBudget(budget)
	selection := BudgetSelection{Budget: budget}
	if len(tools) == 0 {
		return selection
	}

	candidates := make([]Tool, 0, len(tools))
	preOmitted := make([]Tool, 0)
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		tool = normalizeBudgetTool(tool)
		if !isBudgetCandidate(tool) {
			preOmitted = append(preOmitted, tool)
			continue
		}
		identity := budgetToolIdentity(tool)
		if identity != "" {
			if _, ok := seen[identity]; ok {
				continue
			}
			seen[identity] = struct{}{}
		}
		candidates = append(candidates, tool)
	}

	for _, tool := range orderBudgetToolsRoundRobin(candidates) {
		schemaBytes := positiveInt(tool.SchemaBytes)
		estimatedTokens := positiveInt(tool.EstimatedTokens)
		tool.SchemaBytes = schemaBytes
		tool.EstimatedTokens = estimatedTokens
		switch {
		case schemaBytes > budget.MaxSchemaBytesPerTool:
			selection.Omitted = append(selection.Omitted, omitBudgetTool(tool, OmittedReasonSchemaTooLarge))
		case len(selection.Selected) >= budget.MaxTools:
			selection.Omitted = append(selection.Omitted, omitBudgetTool(tool, OmittedReasonToolCountBudgetExceeded))
		case selection.EstimatedTokens+estimatedTokens > budget.EstimatedTokens:
			selection.Omitted = append(selection.Omitted, omitBudgetTool(tool, OmittedReasonTokenBudgetExceeded))
		default:
			tool.Registered = true
			tool.Visible = true
			tool.OmittedReason = ""
			selection.Selected = append(selection.Selected, tool)
			selection.EstimatedTokens += estimatedTokens
		}
	}

	selection.Omitted = append(selection.Omitted, preOmitted...)
	sortBudgetTools(selection.Omitted)
	return selection
}

// AnalysisTools は budget 適用後の selected / omitted tool を Analyze 入力として返す。
func (s BudgetSelection) AnalysisTools() []Tool {
	tools := make([]Tool, 0, len(s.Selected)+len(s.Omitted))
	tools = append(tools, s.Selected...)
	tools = append(tools, s.Omitted...)
	return tools
}

func normalizeBudgetTool(tool Tool) Tool {
	tool.ServerName = strings.TrimSpace(tool.ServerName)
	if tool.ServerName == "" {
		tool.ServerName = "(unknown)"
	}
	tool.ToolName = strings.TrimSpace(tool.ToolName)
	tool.ExportedName = strings.TrimSpace(tool.ExportedName)
	tool.OmittedReason = strings.TrimSpace(tool.OmittedReason)
	tool.SchemaBytes = positiveInt(tool.SchemaBytes)
	tool.EstimatedTokens = positiveInt(tool.EstimatedTokens)
	if !tool.Visible && tool.OmittedReason == "" {
		tool.OmittedReason = "omitted"
	}
	return tool
}

func isBudgetCandidate(tool Tool) bool {
	return (tool.Registered || tool.Visible) && tool.Visible && tool.OmittedReason == ""
}

func omitBudgetTool(tool Tool, reason string) Tool {
	tool.Registered = true
	tool.Visible = false
	tool.OmittedReason = reason
	return tool
}

func budgetToolIdentity(tool Tool) string {
	if tool.ExportedName != "" {
		return tool.ExportedName
	}
	if tool.ServerName == "" && tool.ToolName == "" {
		return ""
	}
	return tool.ServerName + "\x00" + tool.ToolName
}

func orderBudgetToolsRoundRobin(tools []Tool) []Tool {
	if len(tools) == 0 {
		return nil
	}

	byServer := make(map[string][]Tool)
	for _, tool := range tools {
		byServer[tool.ServerName] = append(byServer[tool.ServerName], tool)
	}

	servers := make([]string, 0, len(byServer))
	for server := range byServer {
		servers = append(servers, server)
		sortBudgetTools(byServer[server])
	}
	sort.Strings(servers)

	ordered := make([]Tool, 0, len(tools))
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

func sortBudgetTools(tools []Tool) {
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].ServerName != tools[j].ServerName {
			return tools[i].ServerName < tools[j].ServerName
		}
		if tools[i].ToolName != tools[j].ToolName {
			return tools[i].ToolName < tools[j].ToolName
		}
		if tools[i].ExportedName != tools[j].ExportedName {
			return tools[i].ExportedName < tools[j].ExportedName
		}
		return tools[i].OmittedReason < tools[j].OmittedReason
	})
}

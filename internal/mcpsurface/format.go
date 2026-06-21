package mcpsurface

import (
	"fmt"
	"strconv"
	"strings"
)

// MetricName は tool metric の表示名を返す。
func MetricName(metric ToolMetric) string {
	if strings.TrimSpace(metric.ExportedName) != "" {
		return metric.ExportedName
	}
	return metric.ServerName + "." + metric.ToolName
}

// FormatBytes は MCP tool surface 用の byte 数表示を返す。
func FormatBytes(bytes int) string {
	if bytes >= 1024 && bytes%1024 == 0 {
		return fmt.Sprintf("%d KiB", bytes/1024)
	}
	return fmt.Sprintf("%d bytes", bytes)
}

// FormatBudget は MCP tool surface budget を短い text に整形する。
func FormatBudget(budget Budget) string {
	budget = NormalizeBudget(budget)
	return fmt.Sprintf(
		"max %d tools / %d tokens / %s schema per tool",
		budget.MaxTools,
		budget.EstimatedTokens,
		FormatBytes(budget.MaxSchemaBytesPerTool),
	)
}

// FormatReasonCounts は omitted / hidden 理由の件数を安定した text に整形する。
func FormatReasonCounts(counts []ReasonCount, limit int) string {
	if len(counts) == 0 {
		return "-"
	}
	if limit <= 0 || limit > len(counts) {
		limit = len(counts)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s=%d", counts[i].Reason, counts[i].Count))
	}
	if remaining := len(counts) - limit; remaining > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", remaining))
	}
	return strings.Join(parts, ", ")
}

// IncludeSnippet は tools.include の mcpServers 断片を 1 行で返す。
func IncludeSnippet(recommendation Recommendation) string {
	includeTools := recommendation.IncludeTools
	if len(includeTools) == 0 {
		includeTools = []string{"<needed_tool>"}
	}
	return fmt.Sprintf("%s: {\"tools\": {\"include\": [%s]}}", strconv.Quote(recommendation.ServerName), strings.Join(quoteStrings(includeTools), ", "))
}

// SurfaceBudgetSnippet は ~/.xelyon/config.yaml 用の surface_budget 断片を 1 行で返す。
func SurfaceBudgetSnippet(budget Budget) string {
	budget = NormalizeBudget(budget)
	return fmt.Sprintf(
		"mcp: {surface_budget: {max_tools: %d, estimated_tokens: %d, max_schema_bytes_per_tool: %d}}",
		budget.MaxTools,
		budget.EstimatedTokens,
		budget.MaxSchemaBytesPerTool,
	)
}

func quoteStrings(values []string) []string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return quoted
}

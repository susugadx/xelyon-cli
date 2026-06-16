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

func quoteStrings(values []string) []string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return quoted
}

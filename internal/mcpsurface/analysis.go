// Package mcpsurface は MCP tool surface の安全な集計 DTO を提供する。
package mcpsurface

import (
	"sort"
	"strings"
)

const (
	defaultTopLimit                    = 5
	defaultRecommendationToolLimit     = 5
	defaultRecommendationToolThreshold = 20
)

// Tool は raw schema や description を含まない MCP tool の集計入力を表す。
type Tool struct {
	ServerName      string `json:"server_name"`
	ToolName        string `json:"tool_name"`
	ExportedName    string `json:"exported_name,omitempty"`
	Registered      bool   `json:"registered"`
	Visible         bool   `json:"visible"`
	OmittedReason   string `json:"omitted_reason,omitempty"`
	SchemaBytes     int    `json:"schema_bytes,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens,omitempty"`
}

// Options は MCP tool surface 集計の表示用上限を表す。
type Options struct {
	TopLimit                    int
	RecommendationToolLimit     int
	RecommendationToolThreshold int
}

// Report は MCP tool surface の sanitized な集計結果を表す。
type Report struct {
	TotalTools                 int              `json:"total_tools"`
	RegisteredTools            int              `json:"registered_tools"`
	VisibleTools               int              `json:"visible_tools"`
	OmittedTools               int              `json:"omitted_tools"`
	EstimatedTokens            int              `json:"estimated_tokens,omitempty"`
	SchemaBytes                int              `json:"schema_bytes,omitempty"`
	Servers                    []ServerSummary  `json:"servers,omitempty"`
	OmittedReasons             []ReasonCount    `json:"omitted_reasons,omitempty"`
	LargestSchemaTools         []ToolMetric     `json:"largest_schema_tools,omitempty"`
	HighestEstimatedTokenTools []ToolMetric     `json:"highest_estimated_token_tools,omitempty"`
	Recommendations            []Recommendation `json:"recommendations,omitempty"`
}

// ServerSummary は server 単位の MCP tool surface 集計を表す。
type ServerSummary struct {
	ServerName      string        `json:"server_name"`
	TotalTools      int           `json:"total_tools"`
	RegisteredTools int           `json:"registered_tools"`
	VisibleTools    int           `json:"visible_tools"`
	OmittedTools    int           `json:"omitted_tools"`
	EstimatedTokens int           `json:"estimated_tokens,omitempty"`
	SchemaBytes     int           `json:"schema_bytes,omitempty"`
	OmittedReasons  []ReasonCount `json:"omitted_reasons,omitempty"`
}

// ReasonCount は omitted / hidden 理由ごとの件数を表す。
type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// ToolMetric は schema bytes や estimated tokens の上位 tool を表す。
type ToolMetric struct {
	ServerName      string `json:"server_name"`
	ToolName        string `json:"tool_name"`
	ExportedName    string `json:"exported_name,omitempty"`
	Registered      bool   `json:"registered"`
	Visible         bool   `json:"visible"`
	OmittedReason   string `json:"omitted_reason,omitempty"`
	SchemaBytes     int    `json:"schema_bytes,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens,omitempty"`
}

// Recommendation は tool surface を絞るための設定提案を表す。
type Recommendation struct {
	ServerName   string   `json:"server_name"`
	Reason       string   `json:"reason"`
	IncludeTools []string `json:"include_tools,omitempty"`
}

// Analyze は MCP tool surface を server、理由、重い tool ごとに集計する。
func Analyze(tools []Tool, opts Options) Report {
	opts = normalizeOptions(opts)
	report := Report{TotalTools: len(tools)}
	if len(tools) == 0 {
		return report
	}

	serverByName := make(map[string]*ServerSummary)
	toolNamesByServer := make(map[string][]string)
	reasons := make(map[string]int)
	metrics := make([]ToolMetric, 0, len(tools))

	for _, tool := range tools {
		serverName := strings.TrimSpace(tool.ServerName)
		if serverName == "" {
			serverName = "(unknown)"
		}
		summary := serverByName[serverName]
		if summary == nil {
			summary = &ServerSummary{ServerName: serverName}
			serverByName[serverName] = summary
		}

		estimatedTokens := positiveInt(tool.EstimatedTokens)
		report.EstimatedTokens += estimatedTokens
		summary.EstimatedTokens += estimatedTokens

		summary.TotalTools++
		registered := tool.Registered || tool.Visible
		if registered {
			report.RegisteredTools++
			summary.RegisteredTools++
		}
		if tool.Visible {
			report.VisibleTools++
			summary.VisibleTools++
			toolNamesByServer[serverName] = append(toolNamesByServer[serverName], tool.ToolName)
		} else {
			report.OmittedTools++
			summary.OmittedTools++
			reason := strings.TrimSpace(tool.OmittedReason)
			if reason == "" {
				reason = "omitted"
			}
			reasons[reason]++
			summary.OmittedReasons = incrementReason(summary.OmittedReasons, reason)
		}

		schemaBytes := positiveInt(tool.SchemaBytes)
		report.SchemaBytes += schemaBytes
		summary.SchemaBytes += schemaBytes
		metrics = append(metrics, ToolMetric{
			ServerName:      serverName,
			ToolName:        tool.ToolName,
			ExportedName:    tool.ExportedName,
			Registered:      registered,
			Visible:         tool.Visible,
			OmittedReason:   tool.OmittedReason,
			SchemaBytes:     schemaBytes,
			EstimatedTokens: estimatedTokens,
		})
	}

	report.Servers = sortedServerSummaries(serverByName)
	report.OmittedReasons = sortedReasonCounts(reasons)
	report.LargestSchemaTools = topSchemaTools(metrics, opts.TopLimit)
	report.HighestEstimatedTokenTools = topTokenTools(metrics, opts.TopLimit)
	report.Recommendations = buildRecommendations(report.Servers, toolNamesByServer, opts)
	return report
}

func normalizeOptions(opts Options) Options {
	if opts.TopLimit <= 0 {
		opts.TopLimit = defaultTopLimit
	}
	if opts.RecommendationToolLimit <= 0 {
		opts.RecommendationToolLimit = defaultRecommendationToolLimit
	}
	if opts.RecommendationToolThreshold <= 0 {
		opts.RecommendationToolThreshold = defaultRecommendationToolThreshold
	}
	return opts
}

func positiveInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func incrementReason(reasons []ReasonCount, reason string) []ReasonCount {
	for i := range reasons {
		if reasons[i].Reason == reason {
			reasons[i].Count++
			return reasons
		}
	}
	return append(reasons, ReasonCount{Reason: reason, Count: 1})
}

func sortedServerSummaries(byName map[string]*ServerSummary) []ServerSummary {
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]ServerSummary, 0, len(names))
	for _, name := range names {
		summary := *byName[name]
		sortReasonCounts(summary.OmittedReasons)
		summaries = append(summaries, summary)
	}
	return summaries
}

func sortedReasonCounts(values map[string]int) []ReasonCount {
	counts := make([]ReasonCount, 0, len(values))
	for reason, count := range values {
		counts = append(counts, ReasonCount{Reason: reason, Count: count})
	}
	sortReasonCounts(counts)
	return counts
}

func sortReasonCounts(counts []ReasonCount) {
	sort.SliceStable(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Reason < counts[j].Reason
	})
}

func topSchemaTools(metrics []ToolMetric, limit int) []ToolMetric {
	items := make([]ToolMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric.SchemaBytes > 0 {
			items = append(items, metric)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SchemaBytes != items[j].SchemaBytes {
			return items[i].SchemaBytes > items[j].SchemaBytes
		}
		return toolMetricName(items[i]) < toolMetricName(items[j])
	})
	return limitToolMetrics(items, limit)
}

func topTokenTools(metrics []ToolMetric, limit int) []ToolMetric {
	items := make([]ToolMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric.EstimatedTokens > 0 {
			items = append(items, metric)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].EstimatedTokens != items[j].EstimatedTokens {
			return items[i].EstimatedTokens > items[j].EstimatedTokens
		}
		return toolMetricName(items[i]) < toolMetricName(items[j])
	})
	return limitToolMetrics(items, limit)
}

func limitToolMetrics(items []ToolMetric, limit int) []ToolMetric {
	if len(items) > limit {
		return append([]ToolMetric(nil), items[:limit]...)
	}
	return append([]ToolMetric(nil), items...)
}

func toolMetricName(metric ToolMetric) string {
	if strings.TrimSpace(metric.ExportedName) != "" {
		return metric.ExportedName
	}
	return metric.ServerName + "." + metric.ToolName
}

func buildRecommendations(servers []ServerSummary, toolNamesByServer map[string][]string, opts Options) []Recommendation {
	recommendations := make([]Recommendation, 0)
	for _, server := range servers {
		reason := ""
		switch {
		case server.OmittedTools > 0:
			reason = "tools are omitted; narrow the server tool surface"
		case server.RegisteredTools > opts.RecommendationToolThreshold:
			reason = "server exposes many tools; consider narrowing the server tool surface"
		default:
			continue
		}
		names := uniqueSortedNonEmpty(toolNamesByServer[server.ServerName])
		if len(names) > opts.RecommendationToolLimit {
			names = names[:opts.RecommendationToolLimit]
		}
		recommendations = append(recommendations, Recommendation{
			ServerName:   server.ServerName,
			Reason:       reason,
			IncludeTools: append([]string(nil), names...),
		})
	}
	return recommendations
}

func uniqueSortedNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

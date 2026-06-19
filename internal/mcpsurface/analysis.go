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
	defaultBudgetMaxTools              = 80
	defaultBudgetEstimatedTokens       = 32000
	defaultBudgetMaxSchemaBytesPerTool = 128 * 1024
)

const (
	// OmittedReasonSchemaTooLarge は tool schema が 1 tool あたりの上限を超えたことを表す。
	OmittedReasonSchemaTooLarge = "schema_too_large"
	// OmittedReasonToolCountBudgetExceeded は provider-facing tool 数上限を超えたことを表す。
	OmittedReasonToolCountBudgetExceeded = "tool_count_budget_exceeded"
	// OmittedReasonTokenBudgetExceeded は provider-facing tool 定義の推定 token 上限を超えたことを表す。
	OmittedReasonTokenBudgetExceeded = "token_budget_exceeded"
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
	Budget                      Budget
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
	EffectiveBudget            *Budget          `json:"effective_budget,omitempty"`
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

// Budget は provider-facing MCP tool surface の上限を表す。
type Budget struct {
	MaxTools              int `json:"max_tools"`
	EstimatedTokens       int `json:"estimated_tokens"`
	MaxSchemaBytesPerTool int `json:"max_schema_bytes_per_tool"`
}

// BudgetSelection は budget 適用後の sanitized MCP tool surface を表す。
type BudgetSelection struct {
	Selected        []Tool
	Omitted         []Tool
	EstimatedTokens int
	Budget          Budget
}

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

// Analyze は MCP tool surface を server、理由、重い tool ごとに集計する。
func Analyze(tools []Tool, opts Options) Report {
	opts = normalizeOptions(opts)
	report := Report{TotalTools: len(tools)}
	if !opts.Budget.isZero() {
		budget := NormalizeBudget(opts.Budget)
		report.EffectiveBudget = &budget
	}
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

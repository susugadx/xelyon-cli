// Package mcpsurface は MCP tool surface の安全な集計 DTO を提供する。
package mcpsurface

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

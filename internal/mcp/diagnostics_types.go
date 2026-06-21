package mcp

import "github.com/susugadx/xelyon-cli/internal/mcpsurface"

// DiagnosticStatus は MCP doctor check の結果を表す。
type DiagnosticStatus string

const (
	DiagnosticStatusOK   DiagnosticStatus = "ok"
	DiagnosticStatusWarn DiagnosticStatus = "warn"
	DiagnosticStatusFail DiagnosticStatus = "fail"
)

// DiagnosticCheck は MCP doctor の単一チェック結果を表す。
type DiagnosticCheck struct {
	Name       string           `json:"name"`
	Status     DiagnosticStatus `json:"status"`
	Message    string           `json:"message"`
	Detail     string           `json:"detail,omitempty"`
	Suggestion string           `json:"suggestion,omitempty"`
}

// DiagnosticOptions は MCP doctor の実行オプションを表す。
type DiagnosticOptions struct {
	HomeDir       string
	MCPEnabled    bool
	MCPHeadless   bool
	Connect       bool
	Server        string
	IncludeTools  bool
	SurfaceBudget mcpsurface.Budget
}

// DiagnosticReport は MCP doctor の構造化診断結果を表す。
type DiagnosticReport struct {
	Target          string                   `json:"target"`
	ConfigPath      string                   `json:"config_path"`
	ConfigExists    bool                     `json:"config_exists"`
	RuntimeEnabled  bool                     `json:"runtime_enabled"`
	RuntimeHeadless bool                     `json:"runtime_headless"`
	Connect         bool                     `json:"connect"`
	ServerFilter    string                   `json:"server_filter,omitempty"`
	ServerCount     int                      `json:"server_count"`
	Checks          []DiagnosticCheck        `json:"checks"`
	ToolSurface     *mcpsurface.Report       `json:"tool_surface,omitempty"`
	Servers         []DiagnosticServerReport `json:"servers,omitempty"`
}

// DiagnosticServerReport は MCP server 単位の診断結果を表す。
type DiagnosticServerReport struct {
	Name                            string                      `json:"name"`
	Disabled                        bool                        `json:"disabled"`
	Command                         string                      `json:"command,omitempty"`
	ArgCount                        int                         `json:"arg_count"`
	EnvKeys                         []string                    `json:"env_keys,omitempty"`
	Approval                        string                      `json:"approval"`
	ApprovalValid                   bool                        `json:"approval_valid"`
	ConfiguredStartupTimeoutSeconds int                         `json:"configured_startup_timeout_seconds,omitempty"`
	StartupTimeoutSeconds           int                         `json:"startup_timeout_seconds"`
	StartupTimeoutClamped           bool                        `json:"startup_timeout_clamped,omitempty"`
	ConfiguredToolTimeoutSeconds    int                         `json:"configured_tool_timeout_seconds,omitempty"`
	ToolTimeoutSeconds              int                         `json:"tool_timeout_seconds"`
	ToolTimeoutClamped              bool                        `json:"tool_timeout_clamped,omitempty"`
	Include                         []string                    `json:"include,omitempty"`
	Exclude                         []string                    `json:"exclude,omitempty"`
	ToolApprovalCount               int                         `json:"tool_approval_count,omitempty"`
	Checks                          []DiagnosticCheck           `json:"checks,omitempty"`
	Connection                      *DiagnosticConnectionReport `json:"connection,omitempty"`
	Tools                           []DiagnosticToolReport      `json:"tools,omitempty"`
}

// DiagnosticConnectionReport は --connect 時の initialize/tools-list 診断結果を表す。
type DiagnosticConnectionReport struct {
	Attempted            bool               `json:"attempted"`
	Status               string             `json:"status"`
	RawToolCount         int                `json:"raw_tool_count,omitempty"`
	RegisteredToolCount  int                `json:"registered_tool_count,omitempty"`
	SkippedToolCount     int                `json:"skipped_tool_count,omitempty"`
	FilteredToolCount    int                `json:"filtered_tool_count,omitempty"`
	DeniedToolCount      int                `json:"denied_tool_count,omitempty"`
	CollisionToolCount   int                `json:"collision_tool_count,omitempty"`
	UnknownToolApprovals []string           `json:"unknown_tool_approvals,omitempty"`
	UnknownIncludes      []string           `json:"unknown_includes,omitempty"`
	UnknownExcludes      []string           `json:"unknown_excludes,omitempty"`
	ToolSurface          *mcpsurface.Report `json:"tool_surface,omitempty"`
	Error                string             `json:"error,omitempty"`
}

// DiagnosticToolReport は --tools 指定時の MCP tool 表示項目を表す。
type DiagnosticToolReport struct {
	Name         string `json:"name"`
	ExportedName string `json:"exported_name,omitempty"`
	Approval     string `json:"approval,omitempty"`
	Visible      bool   `json:"visible"`
	HiddenReason string `json:"hidden_reason,omitempty"`
}

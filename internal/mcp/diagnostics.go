package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"github.com/susugadx/xelyon-cli/internal/token"
)

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

// Diagnose は MCP 設定と任意の接続確認を診断する。
func Diagnose(ctx context.Context, opts DiagnosticOptions) DiagnosticReport {
	return diagnose(ctx, opts, os.UserHomeDir)
}

func diagnose(ctx context.Context, opts DiagnosticOptions, userHomeDir func() (string, error)) DiagnosticReport {
	configPath, configPathErr := resolveDiagnosticMCPConfigPath(opts.HomeDir, userHomeDir)
	report := newDiagnosticReport(opts, configPath)
	addDiagnosticRuntimeChecks(&report, opts)
	if configPathErr != nil {
		report.addCheck(DiagnosticStatusFail, "mcp_config", "MCP config path could not be resolved", configPathErr.Error(), "Set HOME to the directory that contains ~/.xelyon/mcp.json")
		return report
	}

	cfg, exists, err := readMCPConfigIfExists(configPath)
	report.ConfigExists = exists
	if err != nil {
		report.addCheck(DiagnosticStatusFail, "mcp_config", "failed to read MCP config", err.Error(), "Fix ~/.xelyon/mcp.json and rerun this command")
		return report
	}
	if !exists {
		report.addCheck(DiagnosticStatusWarn, "mcp_config", "MCP config file does not exist", configPath, "Create ~/.xelyon/mcp.json or run xelyon once to create the disabled sample")
		return report
	}
	if cfg == nil {
		cfg = &Config{}
	}
	report.addCheck(DiagnosticStatusOK, "mcp_config", "MCP config file is readable", configPath, "")

	serverNames := sortedDiagnosticServerNames(cfg.MCPServers, report.ServerFilter)
	report.ServerCount = len(serverNames)
	if report.ServerFilter != "" && len(serverNames) == 0 {
		report.addCheck(DiagnosticStatusFail, "server_filter", "MCP server filter did not match any configured server", report.ServerFilter, "Check --server or ~/.xelyon/mcp.json")
		return report
	}
	if len(serverNames) == 0 {
		report.addCheck(DiagnosticStatusWarn, "servers", "no MCP servers are configured", "", "Add entries under mcpServers in ~/.xelyon/mcp.json")
		return report
	}
	report.addCheck(DiagnosticStatusOK, "servers", "MCP servers are configured", fmt.Sprintf("servers=%d", len(serverNames)), "")

	manager := NewManager()
	manager.SetOutput(io.Discard)
	manager.config = cfg
	defer manager.Close()

	connectionSurfaceTools := make(map[string][]mcpsurface.Tool)
	for _, serverName := range serverNames {
		serverConfig := cfg.MCPServers[serverName]
		serverReport := diagnoseServerConfig(serverName, serverConfig)
		if opts.Connect {
			if tools := diagnoseServerConnection(ctx, manager, serverName, serverConfig, opts.IncludeTools, &serverReport); len(tools) > 0 {
				connectionSurfaceTools[serverName] = tools
			}
		}
		report.Servers = append(report.Servers, serverReport)
	}
	if opts.Connect {
		applyDiagnosticRuntimeToolSurface(&report, manager.GetTools(), opts.SurfaceBudget, opts.IncludeTools, connectionSurfaceTools)
	}

	return report
}

func resolveDiagnosticMCPConfigPath(homeDir string, userHomeDir func() (string, error)) (string, error) {
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		resolved, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("user home directory is unavailable: %w", err)
		}
		homeDir = resolved
	}

	_, configPath, err := resolveMCPConfigPaths(homeDir)
	if err != nil {
		return "", err
	}
	return configPath, nil
}

func newDiagnosticReport(opts DiagnosticOptions, configPath string) DiagnosticReport {
	return DiagnosticReport{
		Target:          "mcp",
		ConfigPath:      configPath,
		RuntimeEnabled:  opts.MCPEnabled,
		RuntimeHeadless: opts.MCPHeadless,
		Connect:         opts.Connect,
		ServerFilter:    strings.TrimSpace(opts.Server),
	}
}

func addDiagnosticRuntimeChecks(report *DiagnosticReport, opts DiagnosticOptions) {
	if !opts.MCPEnabled {
		report.addCheck(DiagnosticStatusWarn, "runtime", "MCP is disabled in config.yaml", "mcp.enabled=false", "Set mcp.enabled=true to use MCP during normal runtime")
	} else {
		report.addCheck(DiagnosticStatusOK, "runtime", "MCP runtime is enabled", fmt.Sprintf("headless=%t", opts.MCPHeadless), "")
	}
	if opts.IncludeTools && !opts.Connect {
		report.addCheck(DiagnosticStatusWarn, "tools", "--tools requires --connect to list MCP tools", "", "Run xelyon doctor mcp --connect --tools")
	}
}

// SummaryStatus はレポート全体の代表 status を返す。
func (r DiagnosticReport) SummaryStatus() DiagnosticStatus {
	if r.HasFailures() {
		return DiagnosticStatusFail
	}
	for _, check := range r.allChecks() {
		if check.Status == DiagnosticStatusWarn {
			return DiagnosticStatusWarn
		}
	}
	return DiagnosticStatusOK
}

// HasFailures はレポートに fail check があるかを返す。
func (r DiagnosticReport) HasFailures() bool {
	for _, check := range r.allChecks() {
		if check.Status == DiagnosticStatusFail {
			return true
		}
	}
	return false
}

func (r *DiagnosticReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func (r DiagnosticReport) allChecks() []DiagnosticCheck {
	checks := append([]DiagnosticCheck{}, r.Checks...)
	for _, server := range r.Servers {
		checks = append(checks, server.Checks...)
	}
	return checks
}

func (r *DiagnosticServerReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func readMCPConfigIfExists(configPath string) (*Config, bool, error) {
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}

	cfg, err := readMCPConfig(configPath)
	if err != nil {
		return nil, true, err
	}
	return cfg, true, nil
}

func sortedDiagnosticServerNames(servers map[string]ServerConfig, filter string) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		if filter != "" && name != filter {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func diagnoseServerConfig(serverName string, serverConfig ServerConfig) DiagnosticServerReport {
	approval, approvalValid := mcpapproval.Normalize(serverConfig.Approval)
	startupSeconds, startupClamped := timeoutDiagnostic(serverConfig.StartupTimeoutSeconds, defaultMCPServerOperationTimeout, maxMCPServerOperationTimeout)
	toolSeconds, toolClamped := timeoutDiagnostic(serverConfig.ToolTimeoutSeconds, defaultMCPToolCallTimeout, maxMCPToolCallTimeout)
	report := DiagnosticServerReport{
		Name:                            serverName,
		Disabled:                        serverConfig.Disabled,
		Command:                         serverConfig.Command,
		ArgCount:                        len(serverConfig.Args),
		EnvKeys:                         sortedMapKeys(serverConfig.Env),
		Approval:                        string(approval),
		ApprovalValid:                   approvalValid,
		ConfiguredStartupTimeoutSeconds: serverConfig.StartupTimeoutSeconds,
		StartupTimeoutSeconds:           startupSeconds,
		StartupTimeoutClamped:           startupClamped,
		ConfiguredToolTimeoutSeconds:    serverConfig.ToolTimeoutSeconds,
		ToolTimeoutSeconds:              toolSeconds,
		ToolTimeoutClamped:              toolClamped,
		Include:                         cloneStrings(serverConfig.includeTools()),
		Exclude:                         cloneStrings(serverConfig.excludeTools()),
		ToolApprovalCount:               len(serverConfig.ToolApprovals),
	}

	if serverConfig.Disabled {
		report.addCheck(DiagnosticStatusWarn, "server", "MCP server is disabled", serverName, "Set disabled=false or remove disabled to enable this server")
		return report
	}

	if err := validateMCPCommand(serverConfig.Command); err != nil {
		report.addCheck(DiagnosticStatusFail, "command", "MCP server command is not allowed", err.Error(), "Use one of: "+allowedMCPCommandsText())
	} else {
		report.addCheck(DiagnosticStatusOK, "command", "MCP server command is allowed", serverConfig.Command, "")
	}

	if !approvalValid {
		report.addCheck(DiagnosticStatusWarn, "approval", "MCP server approval is invalid; using confirm", serverConfig.Approval, "Use confirm, auto, or deny")
	} else {
		report.addCheck(DiagnosticStatusOK, "approval", "MCP server approval is valid", string(approval), "")
	}

	report.addTimeoutCheck("startup_timeout", "startupTimeoutSeconds", serverConfig.StartupTimeoutSeconds, startupSeconds, startupClamped)
	report.addTimeoutCheck("tool_timeout", "toolTimeoutSeconds", serverConfig.ToolTimeoutSeconds, toolSeconds, toolClamped)
	report.addToolApprovalValueChecks(serverConfig.ToolApprovals)
	if len(report.Include) > 0 && len(report.Exclude) > 0 {
		report.addCheck(DiagnosticStatusWarn, "tool_filter", "tools.exclude is ignored because tools.include is set", "include has precedence over exclude", "Remove tools.exclude or move names into tools.include")
	}

	return report
}

func (r *DiagnosticServerReport) addTimeoutCheck(name, field string, configured, effective int, clamped bool) {
	if clamped {
		r.addCheck(DiagnosticStatusWarn, name, field+" exceeds max; using capped timeout", fmt.Sprintf("configured=%ds effective=%ds", configured, effective), "Use a value between 1 and 3600 seconds")
		return
	}
	r.addCheck(DiagnosticStatusOK, name, field+" is valid", fmt.Sprintf("configured=%ds effective=%ds", configured, effective), "")
}

func (r *DiagnosticServerReport) addToolApprovalValueChecks(toolApprovals map[string]string) {
	for _, toolName := range sortedMapKeys(toolApprovals) {
		raw := toolApprovals[toolName]
		if _, valid := mcpapproval.Normalize(raw); !valid {
			r.addCheck(DiagnosticStatusWarn, "tool_approval", "MCP tool approval is invalid; using confirm", fmt.Sprintf("%s=%s", toolName, raw), "Use confirm, auto, or deny")
		}
	}
}

func timeoutDiagnostic(configured int, fallback, max time.Duration) (int, bool) {
	effective := mcpTimeoutDuration(configured, fallback, max)
	return int(effective / time.Second), configured > int(max/time.Second)
}

func diagnoseServerConnection(
	ctx context.Context,
	manager *Manager,
	serverName string,
	serverConfig ServerConfig,
	includeTools bool,
	serverReport *DiagnosticServerReport,
) []mcpsurface.Tool {
	connection := &DiagnosticConnectionReport{Attempted: false, Status: "skipped"}
	serverReport.Connection = connection

	if serverConfig.Disabled {
		connection.Error = "server disabled"
		return nil
	}
	if err := validateMCPCommand(serverConfig.Command); err != nil {
		connection.Error = err.Error()
		return nil
	}

	connection.Attempted = true
	client := manager.newClient()
	session, err := manager.openServerSession(ctx, client, serverConfig)
	if err != nil {
		detail := diagnosticMCPServerErrorDetail("initialize", err)
		connection.Status = "fail"
		connection.Error = detail
		serverReport.addCheck(DiagnosticStatusFail, "connect", "MCP server connection failed", detail, "Check command, args, env, and startupTimeoutSeconds")
		return nil
	}

	listCtx, cancel := mcpServerOperationContext(ctx, serverConfig.startupTimeoutDuration())
	defer cancel()
	toolsResult, err := session.ListTools(listCtx, nil)
	if err != nil {
		_ = session.Close()
		detail := diagnosticMCPServerErrorDetail("tools/list", err)
		connection.Status = "fail"
		connection.Error = detail
		serverReport.addCheck(DiagnosticStatusFail, "tools_list", "MCP server tools/list failed", detail, "Check the MCP server logs and startupTimeoutSeconds")
		return nil
	}
	var listedTools []*sdkmcp.Tool
	if toolsResult != nil {
		listedTools = toolsResult.Tools
	}
	serverTools, summary := manager.buildServerTools(serverName, session, listedTools, serverConfig)

	previous := manager.swapServerSession(serverName, session)
	manager.replaceServerTools(serverName, serverTools)
	if previous != nil && previous != session {
		_ = previous.Close()
	}
	manager.markServerHealthy(serverName)

	connection.Status = "ok"
	connection.RegisteredToolCount = summary.registered
	connection.SkippedToolCount = summary.skipped
	serverReport.addCheck(DiagnosticStatusOK, "connect", "MCP server connected", "", "")
	serverReport.addCheck(DiagnosticStatusOK, "tools_list", "MCP server tools/list succeeded", fmt.Sprintf("registered=%d skipped=%d", summary.registered, summary.skipped), "")

	return populateConnectionDiagnostics(manager, serverName, serverConfig, listedTools, connection, serverReport, includeTools)
}

func diagnosticMCPServerErrorDetail(operation string, err error) string {
	status := operation + " failed"
	if errors.Is(err, context.DeadlineExceeded) {
		status = operation + " timed out"
	} else if errors.Is(err, context.Canceled) {
		status = operation + " canceled"
	}
	return status + "; server error detail suppressed by doctor mcp privacy policy"
}

func populateConnectionDiagnostics(
	manager *Manager,
	serverName string,
	serverConfig ServerConfig,
	listedTools []*sdkmcp.Tool,
	connection *DiagnosticConnectionReport,
	serverReport *DiagnosticServerReport,
	includeTools bool,
) []mcpsurface.Tool {
	rawNames := make(map[string]bool, len(listedTools))
	rawToolCount := 0
	for _, tool := range listedTools {
		if tool != nil {
			rawToolCount++
			rawNames[tool.Name] = true
		}
	}
	connection.RawToolCount = rawToolCount
	connection.UnknownToolApprovals = sortedUnknownKeys(serverConfig.ToolApprovals, rawNames)
	connection.UnknownIncludes = sortedUnknownNames(serverConfig.includeTools(), rawNames)
	if len(serverConfig.includeTools()) == 0 {
		connection.UnknownExcludes = sortedUnknownNames(serverConfig.excludeTools(), rawNames)
	}
	addUnknownToolReferenceChecks(serverReport, connection)

	decisions, _ := manager.planServerToolRegistration(serverName, listedTools, serverConfig)
	toolSurfaceTools := make([]mcpsurface.Tool, 0, len(decisions))
	for _, decision := range decisions {
		switch decision.skipReason {
		case toolSkipFiltered:
			connection.FilteredToolCount++
		case toolSkipServerDeny, toolSkipToolDeny:
			connection.DeniedToolCount++
		case toolSkipCollision:
			connection.CollisionToolCount++
		}
		toolSurfaceTools = append(toolSurfaceTools, diagnosticToolSurfaceTool(serverName, decision))
	}
	if includeTools {
		for _, decision := range decisions {
			serverReport.Tools = append(serverReport.Tools, diagnosticToolReport(decision))
		}
	}
	return toolSurfaceTools
}

func addUnknownToolReferenceChecks(serverReport *DiagnosticServerReport, connection *DiagnosticConnectionReport) {
	if len(connection.UnknownToolApprovals) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_approvals", "toolApprovals contains unknown tool names", strings.Join(connection.UnknownToolApprovals, ", "), "Use raw tool names returned by the MCP server")
	}
	if len(connection.UnknownIncludes) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_filter", "tools.include contains unknown tool names", strings.Join(connection.UnknownIncludes, ", "), "Use raw tool names returned by the MCP server")
	}
	if len(connection.UnknownExcludes) > 0 {
		serverReport.addCheck(DiagnosticStatusWarn, "tool_filter", "tools.exclude contains unknown tool names", strings.Join(connection.UnknownExcludes, ", "), "Use raw tool names returned by the MCP server")
	}
}

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

func sortedUnknownKeys(values map[string]string, known map[string]bool) []string {
	keys := sortedMapKeys(values)
	return sortedUnknownNames(keys, known)
}

func sortedUnknownNames(values []string, known map[string]bool) []string {
	unknown := make([]string, 0)
	for _, value := range values {
		if !known[value] {
			unknown = append(unknown, value)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func (c ServerConfig) includeTools() []string {
	if c.Tools == nil {
		return nil
	}
	return c.Tools.Include
}

func (c ServerConfig) excludeTools() []string {
	if c.Tools == nil {
		return nil
	}
	return c.Tools.Exclude
}

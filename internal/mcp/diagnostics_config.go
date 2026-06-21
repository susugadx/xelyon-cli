package mcp

import (
	"errors"
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"os"
	"sort"
	"strings"
	"time"
)

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

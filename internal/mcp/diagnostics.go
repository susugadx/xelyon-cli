package mcp

import (
	"context"
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
	"io"
	"os"
)

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

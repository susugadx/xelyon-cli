package mcp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func TestDiagnoseToolsWithoutConnectWarns(t *testing.T) {
	homeDir := t.TempDir()
	writeMCPDiagnosticConfig(t, homeDir, Config{
		MCPServers: map[string]ServerConfig{
			"sample": {Command: "npx"},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:      homeDir,
		MCPEnabled:   true,
		IncludeTools: true,
	})

	if report.SummaryStatus() != DiagnosticStatusWarn {
		t.Fatalf("SummaryStatus = %q, want warn", report.SummaryStatus())
	}
	requireMCPDiagnosticCheck(t, report.Checks, "tools", DiagnosticStatusWarn)
	if len(report.Servers) != 1 || report.Servers[0].Connection != nil {
		t.Fatalf("servers = %#v, want local-only server report without connection", report.Servers)
	}
}

func TestDiagnoseConnectReportsToolVisibilityAndUnknownReferences(t *testing.T) {
	command, args := mcpHelperCommand(t)
	homeDir := t.TempDir()
	writeMCPDiagnosticConfig(t, homeDir, Config{
		MCPServers: map[string]ServerConfig{
			"helper": {
				Command: command,
				Args:    args,
				Env: map[string]string{
					"GO_WANT_XELYON_MCP_HELPER": "1",
				},
				Approval: "auto",
				Tools: &ToolsFilter{
					Exclude: []string{"hidden", "missing_exclude"},
				},
				ToolApprovals: map[string]string{
					"echo":         "confirm",
					"missing_tool": "deny",
				},
			},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:      homeDir,
		MCPEnabled:   true,
		Connect:      true,
		IncludeTools: true,
	})

	if report.SummaryStatus() != DiagnosticStatusWarn {
		t.Fatalf("SummaryStatus = %q, want warn for unknown references", report.SummaryStatus())
	}
	if len(report.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(report.Servers))
	}
	server := report.Servers[0]
	if server.Command != command {
		t.Fatalf("Command = %q, want %q", server.Command, command)
	}
	if server.ArgCount == 0 {
		t.Fatal("ArgCount = 0, want helper args counted")
	}
	if len(server.EnvKeys) != 1 || server.EnvKeys[0] != "GO_WANT_XELYON_MCP_HELPER" {
		t.Fatalf("EnvKeys = %#v, want only key names", server.EnvKeys)
	}
	requireMCPDiagnosticCheck(t, server.Checks, "connect", DiagnosticStatusOK)
	requireMCPDiagnosticCheck(t, server.Checks, "tools_list", DiagnosticStatusOK)
	requireMCPDiagnosticCheck(t, server.Checks, "tool_approvals", DiagnosticStatusWarn)
	requireMCPDiagnosticCheck(t, server.Checks, "tool_filter", DiagnosticStatusWarn)
	if server.Connection == nil {
		t.Fatal("Connection = nil, want connection report")
	}
	if server.Connection.Status != "ok" || !server.Connection.Attempted {
		t.Fatalf("Connection = %+v, want attempted ok", *server.Connection)
	}
	if server.Connection.RawToolCount != 2 || server.Connection.RegisteredToolCount != 1 || server.Connection.SkippedToolCount != 1 {
		t.Fatalf("Connection counts = %+v, want raw=2 registered=1 skipped=1", *server.Connection)
	}
	if server.Connection.FilteredToolCount != 1 {
		t.Fatalf("FilteredToolCount = %d, want 1", server.Connection.FilteredToolCount)
	}
	if server.Connection.ToolSurface == nil {
		t.Fatal("ToolSurface = nil, want live tool surface analysis")
	}
	if server.Connection.ToolSurface.TotalTools != 2 || server.Connection.ToolSurface.RegisteredTools != server.Connection.RegisteredToolCount || server.Connection.ToolSurface.VisibleTools != 1 || server.Connection.ToolSurface.OmittedTools != 1 {
		t.Fatalf("ToolSurface counts = %+v, want total=2 registered=%d visible=1 omitted=1", *server.Connection.ToolSurface, server.Connection.RegisteredToolCount)
	}
	if len(server.Connection.ToolSurface.Servers) != 1 {
		t.Fatalf("ToolSurface servers = %#v, want one helper summary", server.Connection.ToolSurface.Servers)
	}
	surfaceServer := server.Connection.ToolSurface.Servers[0]
	if surfaceServer.TotalTools != 2 || surfaceServer.RegisteredTools != 1 || surfaceServer.VisibleTools != 1 || surfaceServer.OmittedTools != 1 {
		t.Fatalf("ToolSurface server counts = %+v, want total=2 registered=1 visible=1 omitted=1", surfaceServer)
	}
	if len(server.Connection.ToolSurface.OmittedReasons) != 1 || server.Connection.ToolSurface.OmittedReasons[0].Reason != string(toolSkipFiltered) {
		t.Fatalf("ToolSurface omitted reasons = %#v, want filtered", server.Connection.ToolSurface.OmittedReasons)
	}
	if len(server.Connection.ToolSurface.Recommendations) != 1 || server.Connection.ToolSurface.Recommendations[0].ServerName != "helper" {
		t.Fatalf("ToolSurface recommendations = %#v, want helper recommendation", server.Connection.ToolSurface.Recommendations)
	}
	if len(server.Connection.UnknownToolApprovals) != 1 || server.Connection.UnknownToolApprovals[0] != "missing_tool" {
		t.Fatalf("UnknownToolApprovals = %#v, want missing_tool", server.Connection.UnknownToolApprovals)
	}
	if len(server.Connection.UnknownExcludes) != 1 || server.Connection.UnknownExcludes[0] != "missing_exclude" {
		t.Fatalf("UnknownExcludes = %#v, want missing_exclude", server.Connection.UnknownExcludes)
	}
	if len(server.Tools) != 2 {
		t.Fatalf("Tools = %#v, want visible and hidden tools", server.Tools)
	}
	if server.Tools[0].Name != "echo" || !server.Tools[0].Visible || server.Tools[0].Approval != "confirm" {
		t.Fatalf("first tool = %+v, want visible echo confirm", server.Tools[0])
	}
	if server.Tools[1].Name != "hidden" || server.Tools[1].Visible || server.Tools[1].HiddenReason != string(toolSkipFiltered) {
		t.Fatalf("second tool = %+v, want filtered hidden", server.Tools[1])
	}
}

func TestPopulateConnectionDiagnosticsMarksDuplicateToolCollisionHidden(t *testing.T) {
	manager := NewManager()
	manager.SetOutput(io.Discard)
	serverConfig := ServerConfig{Approval: "auto"}
	listedTools := []*sdkmcp.Tool{
		{Name: "dup"},
		{Name: "dup"},
	}
	_, summary := manager.buildServerTools("helper", nil, listedTools, serverConfig)
	if summary.registered != 1 || summary.skipped != 1 {
		t.Fatalf("registration summary = %+v, want registered=1 skipped=1", summary)
	}

	connection := &DiagnosticConnectionReport{
		RegisteredToolCount: summary.registered,
		SkippedToolCount:    summary.skipped,
	}
	serverReport := &DiagnosticServerReport{Name: "helper"}
	populateConnectionDiagnostics(manager, "helper", serverConfig, listedTools, connection, serverReport, true)

	if connection.RawToolCount != 2 || connection.RegisteredToolCount != 1 || connection.SkippedToolCount != 1 || connection.CollisionToolCount != 1 {
		t.Fatalf("Connection counts = %+v, want raw=2 registered=1 skipped=1 collision=1", *connection)
	}
	if connection.ToolSurface == nil {
		t.Fatal("ToolSurface = nil, want duplicate visibility analysis")
	}
	if connection.ToolSurface.TotalTools != 2 || connection.ToolSurface.RegisteredTools != connection.RegisteredToolCount || connection.ToolSurface.VisibleTools != 1 || connection.ToolSurface.OmittedTools != 1 {
		t.Fatalf("ToolSurface counts = %+v, want total=2 registered=%d visible=1 omitted=1", *connection.ToolSurface, connection.RegisteredToolCount)
	}
	if len(connection.ToolSurface.Servers) != 1 {
		t.Fatalf("ToolSurface servers = %#v, want one helper summary", connection.ToolSurface.Servers)
	}
	surfaceServer := connection.ToolSurface.Servers[0]
	if surfaceServer.TotalTools != 2 || surfaceServer.RegisteredTools != 1 || surfaceServer.VisibleTools != 1 || surfaceServer.OmittedTools != 1 {
		t.Fatalf("ToolSurface server counts = %+v, want total=2 registered=1 visible=1 omitted=1", surfaceServer)
	}
	if len(connection.ToolSurface.OmittedReasons) != 1 || connection.ToolSurface.OmittedReasons[0].Reason != string(toolSkipCollision) {
		t.Fatalf("ToolSurface omitted reasons = %#v, want collision", connection.ToolSurface.OmittedReasons)
	}
	if len(serverReport.Tools) != 2 {
		t.Fatalf("Tools = %#v, want two duplicate entries", serverReport.Tools)
	}
	first := serverReport.Tools[0]
	if first.Name != "dup" || first.ExportedName != "mcp_helper_dup" || first.Approval != "auto" || !first.Visible || first.HiddenReason != "" {
		t.Fatalf("first tool = %+v, want visible duplicate winner", first)
	}
	second := serverReport.Tools[1]
	if second.Name != "dup" || second.ExportedName != "mcp_helper_dup" || second.Approval != "auto" || second.Visible || second.HiddenReason != string(toolSkipCollision) {
		t.Fatalf("second tool = %+v, want hidden duplicate collision", second)
	}
}

func TestPopulateConnectionDiagnosticsEstimatesTokensFromActualProviderSchemaWithoutLeakingIt(t *testing.T) {
	manager := NewManager()
	manager.SetOutput(io.Discard)
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"payload": map[string]any{
				"type":        "string",
				"description": strings.Repeat("SECRET_SCHEMA ", 30),
			},
		},
	}
	listedTools := []*sdkmcp.Tool{{
		Name:        "heavy",
		Description: "SECRET_DESCRIPTION",
		InputSchema: inputSchema,
	}}
	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		t.Fatalf("json.Marshal(schema) error = %v", err)
	}
	expectedTokens := token.EstimateStructuredValueTokenCount(api.ConvertMCPToolToToolDefinition(
		"mcp_helper_heavy",
		"SECRET_DESCRIPTION",
		schemaBytes,
	))
	connection := &DiagnosticConnectionReport{RegisteredToolCount: 1}
	serverReport := &DiagnosticServerReport{Name: "helper"}

	populateConnectionDiagnostics(manager, "helper", ServerConfig{}, listedTools, connection, serverReport, true)

	if connection.ToolSurface == nil {
		t.Fatal("ToolSurface = nil, want tool surface analysis")
	}
	if len(connection.ToolSurface.HighestEstimatedTokenTools) == 0 {
		t.Fatalf("HighestEstimatedTokenTools = %#v, want heavy tool", connection.ToolSurface.HighestEstimatedTokenTools)
	}
	if got := connection.ToolSurface.HighestEstimatedTokenTools[0].EstimatedTokens; got != expectedTokens {
		t.Fatalf("estimated tokens = %d, want provider-definition estimate %d", got, expectedTokens)
	}
	data, err := json.Marshal(connection)
	if err != nil {
		t.Fatalf("json.Marshal(connection) error = %v", err)
	}
	for _, leaked := range []string{"SECRET_SCHEMA", "SECRET_DESCRIPTION"} {
		if strings.Contains(string(data), leaked) {
			t.Fatalf("diagnostic tool surface leaked %q:\n%s", leaked, string(data))
		}
	}
}

func TestDiagnoseConnectSuppressesServerControlledErrorDetails(t *testing.T) {
	tests := []struct {
		name       string
		failMethod string
		checkName  string
		wantDetail string
	}{
		{
			name:       "initialize",
			failMethod: "initialize",
			checkName:  "connect",
			wantDetail: "initialize failed; server error detail suppressed by doctor mcp privacy policy",
		},
		{
			name:       "tools list",
			failMethod: "tools/list",
			checkName:  "tools_list",
			wantDetail: "tools/list failed; server error detail suppressed by doctor mcp privacy policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, args := mcpHelperCommand(t)
			args = append(args, "--secret=SECRETARG")
			homeDir := t.TempDir()
			t.Setenv("XELYON_TEST_MCP_SECRET_ENV", "SECRETENV")
			writeMCPDiagnosticConfig(t, homeDir, Config{
				MCPServers: map[string]ServerConfig{
					"helper": {
						Command: command,
						Args:    args,
						Env: map[string]string{
							"GO_WANT_XELYON_MCP_HELPER":      "1",
							"GO_WANT_XELYON_MCP_HELPER_FAIL": tt.failMethod,
							"MCP_SECRET_ENV":                 "${XELYON_TEST_MCP_SECRET_ENV}",
						},
					},
				},
			})

			report := Diagnose(context.Background(), DiagnosticOptions{
				HomeDir:    homeDir,
				MCPEnabled: true,
				Connect:    true,
			})

			if report.SummaryStatus() != DiagnosticStatusFail {
				t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
			}
			if len(report.Servers) != 1 {
				t.Fatalf("servers = %d, want 1", len(report.Servers))
			}
			server := report.Servers[0]
			if server.Connection == nil {
				t.Fatal("Connection = nil, want failed connection report")
			}
			if server.Connection.Error != tt.wantDetail {
				t.Fatalf("Connection.Error = %q, want %q", server.Connection.Error, tt.wantDetail)
			}
			check := requireMCPDiagnosticCheck(t, server.Checks, tt.checkName, DiagnosticStatusFail)
			if check.Detail != tt.wantDetail {
				t.Fatalf("check detail = %q, want %q", check.Detail, tt.wantDetail)
			}

			data, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			for _, leaked := range []string{"SECRETARG", "SECRETENV"} {
				if strings.Contains(string(data), leaked) {
					t.Fatalf("diagnostic report leaked %q:\n%s", leaked, string(data))
				}
			}
		})
	}
}

func TestDiagnoseConnectServerDenyHidesAllToolsDespiteToolOverrides(t *testing.T) {
	command, args := mcpHelperCommand(t)
	homeDir := t.TempDir()
	writeMCPDiagnosticConfig(t, homeDir, Config{
		MCPServers: map[string]ServerConfig{
			"helper": {
				Command: command,
				Args:    args,
				Env: map[string]string{
					"GO_WANT_XELYON_MCP_HELPER": "1",
				},
				Approval: "deny",
				ToolApprovals: map[string]string{
					"echo":   "auto",
					"hidden": "confirm",
				},
			},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:      homeDir,
		MCPEnabled:   true,
		Connect:      true,
		IncludeTools: true,
	})

	if report.SummaryStatus() != DiagnosticStatusOK {
		t.Fatalf("SummaryStatus = %q, want ok", report.SummaryStatus())
	}
	server := report.Servers[0]
	if server.Connection == nil {
		t.Fatal("Connection = nil, want connection report")
	}
	if server.Connection.RegisteredToolCount != 0 || server.Connection.SkippedToolCount != 2 || server.Connection.DeniedToolCount != 2 {
		t.Fatalf("Connection counts = %+v, want registered=0 skipped=2 denied=2", *server.Connection)
	}
	if len(server.Tools) != 2 {
		t.Fatalf("Tools = %#v, want two hidden tools", server.Tools)
	}
	for _, tool := range server.Tools {
		if tool.Visible {
			t.Fatalf("tool %+v should be hidden by server deny", tool)
		}
		if tool.HiddenReason != string(toolSkipServerDeny) {
			t.Fatalf("HiddenReason = %q, want %q for %+v", tool.HiddenReason, toolSkipServerDeny, tool)
		}
		if tool.Approval != "deny" {
			t.Fatalf("Approval = %q, want deny for %+v", tool.Approval, tool)
		}
	}
}

func TestDiagnoseServerFilterUnknownFails(t *testing.T) {
	homeDir := t.TempDir()
	writeMCPDiagnosticConfig(t, homeDir, Config{
		MCPServers: map[string]ServerConfig{
			"helper": {Command: "npx"},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:    homeDir,
		MCPEnabled: true,
		Server:     "missing",
	})

	if report.SummaryStatus() != DiagnosticStatusFail {
		t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
	}
	requireMCPDiagnosticCheck(t, report.Checks, "server_filter", DiagnosticStatusFail)
}

package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnoseMissingConfigDoesNotCreateDefault(t *testing.T) {
	homeDir := t.TempDir()

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:    homeDir,
		MCPEnabled: true,
	})

	if report.ConfigExists {
		t.Fatal("ConfigExists = true, want false")
	}
	if report.SummaryStatus() != DiagnosticStatusWarn {
		t.Fatalf("SummaryStatus = %q, want warn", report.SummaryStatus())
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".xelyon", "mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("mcp.json should not be created by doctor: %v", err)
	}
	requireMCPDiagnosticCheck(t, report.Checks, "mcp_config", DiagnosticStatusWarn)
}

func TestDiagnoseUnresolvedHomeDoesNotReadRelativeConfigOrConnect(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	writeMCPDiagnosticConfig(t, cwd, Config{
		MCPServers: map[string]ServerConfig{
			"cwd_server": {
				Command: "npx",
			},
		},
	})

	report := diagnose(context.Background(), DiagnosticOptions{
		MCPEnabled: true,
		Connect:    true,
	}, func() (string, error) {
		return "", errors.New("home unavailable")
	})

	if report.SummaryStatus() != DiagnosticStatusFail {
		t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
	}
	if report.ConfigPath != "" {
		t.Fatalf("ConfigPath = %q, want empty unresolved path", report.ConfigPath)
	}
	if report.ConfigExists {
		t.Fatal("ConfigExists = true, want false because cwd config must not be read")
	}
	if report.ServerCount != 0 || len(report.Servers) != 0 {
		t.Fatalf("servers = count %d report %#v, want no cwd server diagnostics", report.ServerCount, report.Servers)
	}
	check := requireMCPDiagnosticCheck(t, report.Checks, "mcp_config", DiagnosticStatusFail)
	if !strings.Contains(check.Detail, "home unavailable") {
		t.Fatalf("mcp_config detail = %q, want home lookup failure", check.Detail)
	}
}

func TestDiagnoseLocalConfigReportsInvalidCommandWithoutConnect(t *testing.T) {
	homeDir := t.TempDir()
	writeMCPDiagnosticConfig(t, homeDir, Config{
		MCPServers: map[string]ServerConfig{
			"bad": {
				Command:  "sh",
				Approval: "prompt",
			},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:    homeDir,
		MCPEnabled: true,
	})

	if report.SummaryStatus() != DiagnosticStatusFail {
		t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
	}
	if len(report.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(report.Servers))
	}
	server := report.Servers[0]
	requireMCPDiagnosticCheck(t, server.Checks, "command", DiagnosticStatusFail)
	requireMCPDiagnosticCheck(t, server.Checks, "approval", DiagnosticStatusWarn)
	if server.Connection != nil {
		t.Fatalf("Connection = %#v, want nil without --connect", server.Connection)
	}
}

func TestDiagnoseInvalidExistingConfigPreservesConfigExists(t *testing.T) {
	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), []byte("{"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report := Diagnose(context.Background(), DiagnosticOptions{
		HomeDir:    homeDir,
		MCPEnabled: true,
	})

	if !report.ConfigExists {
		t.Fatal("ConfigExists = false, want true for existing invalid mcp.json")
	}
	if report.SummaryStatus() != DiagnosticStatusFail {
		t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
	}
	requireMCPDiagnosticCheck(t, report.Checks, "mcp_config", DiagnosticStatusFail)
}

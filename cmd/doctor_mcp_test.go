package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	mcpdiag "github.com/susugadx/xelyon-cli/internal/mcp"
)

func TestMCPDoctorCommandFlags(t *testing.T) {
	cmd, _ := newDoctorSubcommandTest(t, newMCPDoctorCommand)
	requireDoctorCommandFlags(t, cmd, []string{"connect", "server", "tools", "json"})
	requireDoctorCommandOmitsFlags(t, cmd, []string{
		"model",
		"catalog-model",
		"smoke",
		"tool-smoke",
		"print-request",
		"capabilities",
		"require-capability",
	})
}

func TestRunMCPDoctorInvocation_LocalJSONDoesNotCreateMCPConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cmd, out := newDoctorSubcommandTest(t, newMCPDoctorCommand)
	doctorJSONFlag = true

	if err := runMCPDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runMCPDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[mcpdiag.DiagnosticReport](t, out)
	if report.Target != "mcp" {
		t.Fatalf("Target = %q, want mcp", report.Target)
	}
	if report.ConfigExists {
		t.Fatal("ConfigExists = true, want false")
	}
	if report.Connect {
		t.Fatal("Connect = true, want false by default")
	}
	requireMCPDoctorJSONCheck(t, report.Checks, "mcp_config", "warn")
	if _, err := os.Stat(filepath.Join(homeDir, ".xelyon", "mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("mcp.json should not be created by doctor mcp: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".xelyon", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("config.yaml should not be created by doctor mcp: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, ".xelyon", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should not be created by doctor mcp: %v", err)
	}
}

func TestRunMCPDoctorInvocation_InvalidCommandFails(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	writeMCPDoctorCommandConfig(t, homeDir, map[string]any{
		"mcpServers": map[string]any{
			"bad": map[string]any{
				"command": "sh",
			},
		},
	})

	cmd, out := newDoctorSubcommandTest(t, newMCPDoctorCommand)
	doctorJSONFlag = true

	err := runMCPDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runMCPDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[mcpdiag.DiagnosticReport](t, out)
	if report.SummaryStatus() != mcpdiag.DiagnosticStatusFail {
		t.Fatalf("SummaryStatus = %q, want fail", report.SummaryStatus())
	}
	if len(report.Servers) != 1 {
		t.Fatalf("servers = %d, want 1", len(report.Servers))
	}
	requireMCPDoctorJSONCheck(t, report.Servers[0].Checks, "command", "fail")
}

func TestRunMCPDoctorInvocation_RendersText(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	writeMCPDoctorCommandConfig(t, homeDir, map[string]any{
		"mcpServers": map[string]any{
			"sample": map[string]any{
				"command":  "npx",
				"args":     []string{"--token=RAW_ARG_SECRET"},
				"env":      map[string]string{"ZETA_TOKEN": "SECRET_VALUE", "ALPHA_KEY": "OTHER_SECRET"},
				"approval": "confirm",
			},
		},
	})

	cmd, out := newDoctorSubcommandTest(t, newMCPDoctorCommand)

	if err := runMCPDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runMCPDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	text := out.String()
	for _, want := range []string{
		"MCP doctor",
		"Status: OK",
		"Runtime: enabled=true headless=false",
		"- sample: disabled=false command=npx args=1 env_keys=2",
		"  env keys: ALPHA_KEY, ZETA_TOKEN",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output = %q, want substring %q", text, want)
		}
	}
	for _, leaked := range []string{"SECRET_VALUE", "OTHER_SECRET", "RAW_ARG_SECRET"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("output leaked %q:\n%s", leaked, text)
		}
	}
}

func TestRenderMCPDoctorTextMarksUnresolvedConfigPath(t *testing.T) {
	var out strings.Builder

	renderMCPDoctorText(&out, mcpdiag.DiagnosticReport{
		Target:         "mcp",
		RuntimeEnabled: true,
		Connect:        true,
		Checks: []mcpdiag.DiagnosticCheck{
			{
				Name:    "mcp_config",
				Status:  mcpdiag.DiagnosticStatusFail,
				Message: "MCP config path could not be resolved",
			},
		},
	})

	text := out.String()
	if !strings.Contains(text, "Config: (unresolved) (missing)") {
		t.Fatalf("output = %q, want unresolved config path", text)
	}
}

func TestNewDoctorCommandIncludesMCP(t *testing.T) {
	doctor := newDoctorCommand()
	if findSubcommand(doctor, "mcp") == nil {
		t.Fatal("doctor command missing mcp subcommand")
	}
}

func TestMCPDoctorDocs(t *testing.T) {
	commandsDoc := readRepoText(t, filepath.Join("docs", "commands.md"))
	for _, want := range []string{
		"### `xelyon doctor mcp`",
		"xelyon doctor mcp --connect --tools",
		"`doctor mcp` は env value と raw args を出力しません",
	} {
		if !strings.Contains(commandsDoc, want) {
			t.Fatalf("docs/commands.md missing %q", want)
		}
	}

	mcpDoc := readRepoText(t, filepath.Join("docs", "mcp.md"))
	for _, want := range []string{
		"xelyon doctor mcp",
		"MCP server process も起動しない",
		"tools/call は実行しない",
	} {
		if !strings.Contains(mcpDoc, want) {
			t.Fatalf("docs/mcp.md missing %q", want)
		}
	}
}

func writeMCPDoctorCommandConfig(t *testing.T, homeDir string, payload map[string]any) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func requireMCPDoctorJSONCheck(t *testing.T, checks []mcpdiag.DiagnosticCheck, name, status string) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			if string(check.Status) != status {
				t.Fatalf("check %s status = %q, want %q", name, check.Status, status)
			}
			return
		}
	}
	t.Fatalf("check %s not found in %#v", name, checks)
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

package clidoctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpdiag "github.com/susugadx/xelyon-cli/internal/mcp"
	"github.com/susugadx/xelyon-cli/internal/mcpsurface"
)

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

func TestRenderMCPDoctorTextIncludesToolSurfaceSummary(t *testing.T) {
	var out strings.Builder

	renderMCPDoctorText(&out, mcpdiag.DiagnosticReport{
		Target:         "mcp",
		RuntimeEnabled: true,
		ConfigExists:   true,
		Connect:        true,
		Checks: []mcpdiag.DiagnosticCheck{{
			Name:    "mcp_config",
			Status:  mcpdiag.DiagnosticStatusOK,
			Message: "ok",
		}},
		Servers: []mcpdiag.DiagnosticServerReport{{
			Name: "sample",
			Connection: &mcpdiag.DiagnosticConnectionReport{
				Attempted:           true,
				Status:              "ok",
				RawToolCount:        2,
				RegisteredToolCount: 1,
				SkippedToolCount:    1,
				ToolSurface: &mcpsurface.Report{
					TotalTools:      2,
					RegisteredTools: 1,
					VisibleTools:    1,
					OmittedTools:    1,
					EstimatedTokens: 12,
					SchemaBytes:     34,
					OmittedReasons: []mcpsurface.ReasonCount{{
						Reason: "token_budget_exceeded",
						Count:  1,
					}},
					LargestSchemaTools: []mcpsurface.ToolMetric{{
						ExportedName: "mcp_sample_heavy",
						SchemaBytes:  34,
					}},
					HighestEstimatedTokenTools: []mcpsurface.ToolMetric{{
						ExportedName:    "mcp_sample_heavy",
						EstimatedTokens: 12,
					}},
					Recommendations: []mcpsurface.Recommendation{{
						ServerName:   "sample",
						Reason:       "tools are omitted; narrow the server tool surface",
						IncludeTools: []string{"safe"},
					}},
				},
			},
		}},
	})

	text := out.String()
	for _, want := range []string{
		"tool surface: visible=1 registered=1 total=2 omitted=1 estimated_tokens=12 schema=34 bytes",
		"top omitted reasons: token_budget_exceeded=1",
		"largest schema tools:",
		"mcp_sample_heavy: 34 bytes schema",
		"highest estimated token tools:",
		"mcp_sample_heavy: 12 tokens",
		"recommendations:",
		"\"sample\": {\"tools\": {\"include\": [\"safe\"]}}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output = %q, want substring %q", text, want)
		}
	}
}

func writeMCPDoctorCommandConfig(t *testing.T, homeDir string, payload map[string]any) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), data, 0o644); err != nil {
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

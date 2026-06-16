package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeMCPDiagnosticConfig(t *testing.T, homeDir string, cfg Config) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func requireMCPDiagnosticCheck(t *testing.T, checks []DiagnosticCheck, name string, status DiagnosticStatus) DiagnosticCheck {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			if check.Status != status {
				t.Fatalf("check %s status = %q, want %q", name, check.Status, status)
			}
			return check
		}
	}
	t.Fatalf("check %s not found in %#v", name, checks)
	return DiagnosticCheck{}
}

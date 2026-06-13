package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleInitCommand_UsesRuntimeUI(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	existingPath := filepath.Join(tmpDir, "AGENTS.md")
	const existing = "existing: true\n"
	if err := os.WriteFile(existingPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("n\n"), &out, &out),
		},
	}

	if !handleInitCommand(agent) {
		t.Fatal("handleInitCommand() = false, want true")
	}

	data, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != existing {
		t.Fatalf("xelyon.yaml was overwritten unexpectedly: %q", string(data))
	}

	output := out.String()
	if strings.Contains(output, "Overwrite?") {
		t.Fatalf("did not expect overwrite prompt, got %q", output)
	}
	if !strings.Contains(output, "AGENTS.md already exists. Left unchanged.") {
		t.Fatalf("expected injected output to contain no-overwrite message, got %q", output)
	}
}

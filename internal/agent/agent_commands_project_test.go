package agent

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleProjectCommand_UsesRuntimeOutputWhenConfigMissing(t *testing.T) {
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

	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleProjectCommand(agent) {
		t.Fatal("handleProjectCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "xelyon.yaml not found") {
		t.Fatalf("expected runtime output to contain missing project config message, got %q", output)
	}
	if !strings.Contains(output, "Run /init to create a template") {
		t.Fatalf("expected runtime output to contain init hint, got %q", output)
	}
}

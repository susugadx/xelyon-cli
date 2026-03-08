package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandlePasteCommand_UsesRuntimeOutputOnCancel(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("/c\n"), &out, &out),
		},
	}

	if !handlePasteCommand(agent, nil) {
		t.Fatal("handlePasteCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Paste Mode") {
		t.Fatalf("expected runtime output to contain paste banner, got %q", output)
	}
	if !strings.Contains(output, "Cancelled - input discarded") {
		t.Fatalf("expected runtime output to contain cancel message, got %q", output)
	}
}

package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleLSPCommand_UsesRuntimeOutputWhenDisabled(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	result := handleLSPCommand(agent, nil)
	if !result {
		t.Fatal("handleLSPCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "LSP is not enabled.") {
		t.Fatalf("expected runtime output to contain disabled message, got %q", output)
	}
	if !strings.Contains(output, "lsp.enabled: true") {
		t.Fatalf("expected runtime output to contain enable hint, got %q", output)
	}
}

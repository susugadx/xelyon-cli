package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestConfirmPlan_UsesRuntimePromptIO(t *testing.T) {
	var out bytes.Buffer

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("2\n"), &out, &out),
		},
	}

	approved, feedback := agent.confirmPlan()
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if !strings.Contains(out.String(), "Approve this plan?") {
		t.Fatalf("expected runtime output to contain confirmation prompt, got %q", out.String())
	}
}

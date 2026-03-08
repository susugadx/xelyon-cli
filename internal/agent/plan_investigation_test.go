package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRunInvestigationPhase_DebugOutputUsesRuntimeErrorOutput(t *testing.T) {
	t.Setenv("XELYON_DEBUG_PARSE", "1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	agent := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &errOut),
		},
	}

	plan, err := agent.runInvestigationPhase(context.Background())
	if err != nil {
		t.Fatalf("runInvestigationPhase() error = %v", err)
	}
	if plan != nil {
		t.Fatalf("runInvestigationPhase() plan = %v, want nil", plan)
	}

	if !strings.Contains(errOut.String(), "ParseToolCalls returned 0 tools") {
		t.Fatalf("expected runtime error output to contain debug parse message, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), "mock response") {
		t.Fatalf("expected runtime output to contain assistant response, got %q", out.String())
	}
}

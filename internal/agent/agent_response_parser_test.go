package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleNormalResponse_UsesRuntimeOutputForCompactionNotice(t *testing.T) {
	var outA bytes.Buffer
	var outB bytes.Buffer

	agentA := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &outA, &outA),
		},
	}

	agentA.handleNormalResponse("before [COMPACTION]hidden[/COMPACTION] after")

	if !strings.Contains(outA.String(), "Context compacted by Claude") {
		t.Fatalf("expected runtime output to contain compaction notice, got %q", outA.String())
	}
	if outB.Len() != 0 {
		t.Fatalf("expected other runtime output to stay empty, got %q", outB.String())
	}
	if got := agentA.lastOutputs[len(agentA.lastOutputs)-1]; got != "before  after" {
		t.Fatalf("last output = %q, want %q", got, "before  after")
	}
}

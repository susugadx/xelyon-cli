package subagent

import (
	"context"
	"testing"
)

func TestEmitEvent_NilChannel(t *testing.T) {
	ctx := context.Background()
	EmitEvent(ctx, SubAgentEvent{Tool: "test"})
}

func TestWithEventChannel_RoundTrip(t *testing.T) {
	ch := make(chan SubAgentEvent, 1)
	ctx := WithEventChannel(context.Background(), ch)
	ctx = WithAgentID(ctx, "sub-001")

	EmitEvent(ctx, SubAgentEvent{Tool: "read_file", Phase: "start"})

	event := <-ch
	if event.Tool != "read_file" {
		t.Errorf("Tool = %q, want read_file", event.Tool)
	}
	if event.AgentID != "sub-001" {
		t.Errorf("AgentID = %q, want sub-001", event.AgentID)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

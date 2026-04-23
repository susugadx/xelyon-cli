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

func TestEmitCompletionEvent_Completed(t *testing.T) {
	ch := make(chan SubAgentEvent, 1)
	ctx := WithEventChannel(context.Background(), ch)
	ctx = WithAgentID(ctx, "sub-001")

	EmitCompletionEvent(ctx, "completed", &RunResult{
		Status:         "completed",
		ToolExecutions: 2,
		DurationMs:     1500,
	})

	event := <-ch
	if event.Tool != "_completed" || event.Phase != "end" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if !event.Success {
		t.Fatal("completion event should mark success=true")
	}
	if event.Output != "completed (2 tools, 1.5s)" {
		t.Fatalf("event.Output = %q, want %q", event.Output, "completed (2 tools, 1.5s)")
	}
}

func TestEmitCompletionEvent_ErrorUsesMessage(t *testing.T) {
	ch := make(chan SubAgentEvent, 1)
	ctx := WithEventChannel(context.Background(), ch)
	ctx = WithAgentID(ctx, "sub-001")

	EmitCompletionEvent(ctx, "error", &RunResult{
		Status:       "error",
		ErrorMessage: "boom",
	})

	event := <-ch
	if event.Success {
		t.Fatal("completion event should mark success=false")
	}
	if event.Output != "boom" {
		t.Fatalf("event.Output = %q, want %q", event.Output, "boom")
	}
}

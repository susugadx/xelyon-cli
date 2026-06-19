package toolruntime

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestPlanParallelCallsSkipAndLoopAbortPrecedence(t *testing.T) {
	state := NewParallelCallState([]*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "search_code", Args: map[string]string{"pattern": "skip"}},
		{Tool: "search_code", Args: map[string]string{"pattern": "loop"}},
		{Tool: "read_file", Args: map[string]string{"path": "after.go"}},
	})

	PlanParallelCalls(state,
		func(tc *tools.ToolCall) bool {
			return tc.Args["pattern"] == "loop"
		},
		func(tc *tools.ToolCall) (bool, string) {
			if tc.Args["pattern"] == "skip" {
				return true, "cached"
			}
			return false, ""
		},
	)

	if state.Entries[0].Status != ParallelCallStatusExecute {
		t.Fatalf("entry 0 = %#v, want execute", state.Entries[0])
	}
	if state.Entries[1].Status != ParallelCallStatusSkip || state.Entries[1].SkipMsg != "cached" {
		t.Fatalf("entry 1 = %#v, want skip with message", state.Entries[1])
	}
	if state.Entries[2].Status != ParallelCallStatusLoopAbort || !state.LoopDetected || state.LoopTriggerIdx != 2 {
		t.Fatalf("loop entry/state = %#v detected=%v trigger=%d, want loop abort at 2", state.Entries[2], state.LoopDetected, state.LoopTriggerIdx)
	}
	if state.Entries[3].Status != ParallelCallStatusLoopAbort {
		t.Fatalf("entry 3 = %#v, want abort after loop trigger", state.Entries[3])
	}
}

func TestPartitionParallelAndSequentialUsesExecuteEntriesOnly(t *testing.T) {
	state := NewParallelCallState([]*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "write_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "search_code", Args: map[string]string{"pattern": "skip"}},
	})
	state.Entries[0] = ParallelCallEntry{Status: ParallelCallStatusExecute}
	state.Entries[1] = ParallelCallEntry{Status: ParallelCallStatusExecute}
	state.Entries[2] = ParallelCallEntry{Status: ParallelCallStatusSkip}

	PartitionParallelAndSequential(state)

	if !equalInts(state.ParallelEntries, []int{0}) {
		t.Fatalf("parallel entries = %#v, want [0]", state.ParallelEntries)
	}
	if !equalInts(state.SequentialEntries, []int{1}) {
		t.Fatalf("sequential entries = %#v, want [1]", state.SequentialEntries)
	}
}

func TestBuildLoopAbortHistoryMessageUsesFunctionCallingWhenToolCallIDPresent(t *testing.T) {
	trigger := &tools.ToolCall{ID: "call-loop", Tool: "search_code"}
	msg, ok := BuildLoopAbortHistoryMessage(trigger, 2, 2, 3)
	if !ok {
		t.Fatal("BuildLoopAbortHistoryMessage() ok = false, want true")
	}
	if msg.Role != "tool" || msg.ToolCallID != "call-loop" || msg.ToolName != "search_code" || !strings.Contains(msg.Content, "Tool loop detected") {
		t.Fatalf("trigger message = %#v, want tool loop detection message", msg)
	}

	after := &tools.ToolCall{ID: "call-after", Tool: "read_file"}
	msg, ok = BuildLoopAbortHistoryMessage(after, 3, 2, 3)
	if !ok || msg.Role != "tool" || msg.ToolCallID != "call-after" || msg.Content != "Skipped because a previous tool call in this batch triggered loop detection." {
		t.Fatalf("post-trigger message = %#v ok=%v, want skipped tool message", msg, ok)
	}

	textOnly := &tools.ToolCall{Tool: "search_code"}
	msg, ok = BuildLoopAbortHistoryMessage(textOnly, 2, 2, 3)
	if !ok || msg.Role != "user" || !strings.Contains(msg.Content, "Tool loop detected") || strings.Contains(msg.Content, "[SYSTEM") {
		t.Fatalf("text-only trigger message = %#v ok=%v, want user warning", msg, ok)
	}
}

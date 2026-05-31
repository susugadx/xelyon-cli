package tuiagent

import (
	"context"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

func TestReviewProgressToolResultMapsProgressContractToTUITool(t *testing.T) {
	event := review.ReviewProgressEvent{
		ID:       "evidence",
		Phase:    review.ReviewProgressPhaseEvidence,
		Status:   review.ReviewProgressOK,
		Label:    "evidence collected",
		Detail:   "staged 2 · unstaged 1",
		Duration: 120 * time.Millisecond,
	}

	tool, ok := reviewProgressToolResult(event)
	if !ok {
		t.Fatal("reviewProgressToolResult() ok = false, want true")
	}
	if tool.ID != "review:evidence" {
		t.Fatalf("ID = %q, want review:evidence", tool.ID)
	}
	if tool.Name != "evidence collected" || tool.Target != "· staged 2 · unstaged 1" {
		t.Fatalf("display fields = name:%q target:%q", tool.Name, tool.Target)
	}
	if tool.Status != tui.ToolStatusOK || tool.Error || tool.NonBlockingError {
		t.Fatalf("status fields = status:%q error:%t nonblocking:%t", tool.Status, tool.Error, tool.NonBlockingError)
	}
	if tool.Duration != 120*time.Millisecond || !tool.Collapsed {
		t.Fatalf("render fields = duration:%s collapsed:%t", tool.Duration, tool.Collapsed)
	}
}

func TestReviewProgressToolResultMarksProbeErrorsNonBlocking(t *testing.T) {
	event := review.ReviewProgressEvent{
		ID:     "probe:probe-1:0",
		Phase:  review.ReviewProgressPhaseProbe,
		Status: review.ReviewProgressError,
		Label:  "probe host_readonly",
		Detail: "go test ./internal/tui",
	}

	tool, ok := reviewProgressToolResult(event)
	if !ok {
		t.Fatal("reviewProgressToolResult() ok = false, want true")
	}
	if tool.Status != tui.ToolStatusError || !tool.Error {
		t.Fatalf("tool error status = %q error=%t, want error", tool.Status, tool.Error)
	}
	if !tool.NonBlockingError {
		t.Fatal("probe progress errors should not block the parent review activity")
	}
}

func TestReviewProgressToolResultKeepsNonProbeErrorsBlocking(t *testing.T) {
	event := review.ReviewProgressEvent{
		ID:     "report",
		Phase:  review.ReviewProgressPhaseReport,
		Status: review.ReviewProgressError,
		Label:  "writing report",
		Detail: "provider failed",
	}

	tool, ok := reviewProgressToolResult(event)
	if !ok {
		t.Fatal("reviewProgressToolResult() ok = false, want true")
	}
	if tool.Status != tui.ToolStatusError || !tool.Error {
		t.Fatalf("tool error status = %q error=%t, want error", tool.Status, tool.Error)
	}
	if tool.NonBlockingError {
		t.Fatal("non-probe progress errors should remain blocking")
	}
}

func TestReviewProgressSinkWithRunIDDoesNotFallbackToUnscopedTool(t *testing.T) {
	var toolMessages []tui.AppendToolResultMsg
	adapter := &TUIAdapter{
		sendToolResult: func(msg tui.AppendToolResultMsg) {
			toolMessages = append(toolMessages, msg)
		},
	}

	sink := adapter.reviewProgressSink(tui.ContextWithReviewRunID(context.Background(), 7))
	if sink == nil {
		t.Fatal("reviewProgressSink() = nil, want sink")
	}
	sink(review.ReviewProgressEvent{
		ID:     "probe:probe-1:0",
		Phase:  review.ReviewProgressPhaseProbe,
		Status: review.ReviewProgressRunning,
		Label:  "probe host_readonly",
	})

	if len(toolMessages) != 0 {
		t.Fatalf("unscoped tool messages = %#v, want none for run-scoped progress without review progress sender", toolMessages)
	}
}

func TestReviewProgressSinkWithoutRunIDUsesToolFallback(t *testing.T) {
	var toolMessages []tui.AppendToolResultMsg
	adapter := &TUIAdapter{
		sendToolResult: func(msg tui.AppendToolResultMsg) {
			toolMessages = append(toolMessages, msg)
		},
	}

	sink := adapter.reviewProgressSink(context.Background())
	if sink == nil {
		t.Fatal("reviewProgressSink() = nil, want sink")
	}
	sink(review.ReviewProgressEvent{
		ID:     "evidence",
		Phase:  review.ReviewProgressPhaseEvidence,
		Status: review.ReviewProgressRunning,
		Label:  "collecting current changes",
	})

	if len(toolMessages) != 1 {
		t.Fatalf("tool messages len = %d, want 1", len(toolMessages))
	}
	if got := toolMessages[0].Tool.ID; got != "review:evidence" {
		t.Fatalf("tool ID = %q, want review:evidence", got)
	}
}

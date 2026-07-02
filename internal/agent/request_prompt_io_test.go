package agent

import (
	"context"
	"testing"
	"time"
)

type requestPromptContextKey struct{}

func TestBeginRequestPromptCancellationScope_IgnoresRequestDeadline(t *testing.T) {
	requestCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), requestPromptContextKey{}, "request-value"),
		time.Nanosecond,
	)
	defer cancel()
	<-requestCtx.Done()

	agent := &Agent{}
	promptCtx, cleanup := agent.beginRequestPromptCancellationScope(requestCtx)
	defer cleanup()

	if _, ok := promptCtx.Deadline(); ok {
		t.Fatal("prompt context should not inherit request deadline")
	}
	if err := promptCtx.Err(); err != nil {
		t.Fatalf("prompt context should stay active after request deadline, got %v", err)
	}
	if got := promptCtx.Value(requestPromptContextKey{}); got != "request-value" {
		t.Fatalf("prompt context value = %v, want request-value", got)
	}
	select {
	case <-promptCtx.Done():
		t.Fatal("prompt context should not be cancelled by request deadline")
	default:
	}
}

func TestBeginRequestPromptCancellationScope_ActiveRequestCancelCancelsPromptAndRestoresPrevious(t *testing.T) {
	previousCancelled := false
	agent := &Agent{
		agentRequestState: agentRequestState{
			cancelFunc: func() {
				previousCancelled = true
			},
		},
	}

	promptCtx, cleanup := agent.beginRequestPromptCancellationScope(context.Background())
	agent.cancelActiveRequest("prompt cancelled")

	select {
	case <-promptCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("prompt context was not cancelled by active request cancel")
	}
	if previousCancelled {
		t.Fatal("previous cancel func should not be called while prompt context is active")
	}

	cleanup()
	agent.cancelActiveRequest("after prompt")
	if !previousCancelled {
		t.Fatal("previous cancel func was not restored after prompt cleanup")
	}
}

func TestRequestToolPromptContext_IgnoresRequestDeadlineAndUsesExplicitCancel(t *testing.T) {
	requestCtx, cancelRequest := context.WithTimeout(
		context.WithValue(context.Background(), requestPromptContextKey{}, "tool-request"),
		time.Nanosecond,
	)
	defer cancelRequest()
	<-requestCtx.Done()

	explicitCancelCtx, cancelExplicit := context.WithCancel(context.Background())
	defer cancelExplicit()

	agent := &Agent{
		agentRequestState: agentRequestState{
			requestPromptCancelCtx: explicitCancelCtx,
		},
	}
	promptCtx := agent.requestToolPromptContext(requestCtx)

	if _, ok := promptCtx.Deadline(); ok {
		t.Fatal("tool prompt context should not inherit request deadline")
	}
	if err := promptCtx.Err(); err != nil {
		t.Fatalf("tool prompt context should stay active after request deadline, got %v", err)
	}
	if got := promptCtx.Value(requestPromptContextKey{}); got != "tool-request" {
		t.Fatalf("tool prompt context marker = %v, want tool-request", got)
	}
	select {
	case <-promptCtx.Done():
		t.Fatal("tool prompt context should not be cancelled by request deadline")
	default:
	}

	cancelExplicit()

	select {
	case <-promptCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("tool prompt context should be cancelled by explicit request cancel")
	}
	if err := promptCtx.Err(); err != context.Canceled {
		t.Fatalf("tool prompt context err = %v, want context.Canceled", err)
	}
}

func TestFinalCheckParentContext_IgnoresRequestDeadlineAndUsesExplicitCancel(t *testing.T) {
	requestCtx, cancelRequest := context.WithTimeout(
		context.WithValue(context.Background(), requestPromptContextKey{}, "final-check-request"),
		time.Nanosecond,
	)
	defer cancelRequest()
	<-requestCtx.Done()

	explicitCancelCtx, cancelExplicit := context.WithCancel(context.Background())
	defer cancelExplicit()

	agent := &Agent{
		agentRequestState: agentRequestState{
			requestPromptCancelCtx: explicitCancelCtx,
		},
	}
	finalCheckCtx := agent.finalCheckParentContext(requestCtx)

	if _, ok := finalCheckCtx.Deadline(); ok {
		t.Fatal("final check context should not inherit request deadline")
	}
	if err := finalCheckCtx.Err(); err != nil {
		t.Fatalf("final check context should stay active after request deadline, got %v", err)
	}
	if got := finalCheckCtx.Value(requestPromptContextKey{}); got != "final-check-request" {
		t.Fatalf("final check context marker = %v, want final-check-request", got)
	}
	select {
	case <-finalCheckCtx.Done():
		t.Fatal("final check context should not be cancelled by request deadline")
	default:
	}

	cancelExplicit()

	select {
	case <-finalCheckCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("final check context should be cancelled by explicit request cancel")
	}
	if err := finalCheckCtx.Err(); err != context.Canceled {
		t.Fatalf("final check context err = %v, want context.Canceled", err)
	}
}

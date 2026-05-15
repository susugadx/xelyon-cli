package agent

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func newTurnLoopTestRunner(t *testing.T, extraTools ...tools.Tool) *TurnRunner {
	t.Helper()

	var out bytes.Buffer
	agent := newTurnRunnerTestAgent(
		&sequenceMockProvider{name: "test"},
		newProjectMapDisabledConfig(),
		"",
		&out,
		extraTools...,
	)
	return newTurnRunner(agent, context.Background())
}

func TestRunTurnLoop_AfterPrepareContinueSkipsNoToolHandler(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t)
	requests := 0
	noToolCalls := 0

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 4,
		requestResponse: func(iteration int) (string, error) {
			requests++
			return "no tool response", nil
		},
		afterPrepare: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			if iteration == 0 {
				return turnLoopContinue, nil
			}
			return turnLoopProceed, nil
		},
		onNoToolCalls: func(iteration int, response string) (turnLoopDirective, error) {
			noToolCalls++
			return turnLoopBreak, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopBreak {
		t.Fatalf("directive = %v, want %v", directive, turnLoopBreak)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
	if noToolCalls != 1 {
		t.Fatalf("onNoToolCalls count = %d, want 1", noToolCalls)
	}
}

func TestRunTurnLoop_NoToolBreakDirectiveStopsLoop(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t)
	requests := 0
	beforeCalled := false
	executeCalled := false

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 2,
		requestResponse: func(iteration int) (string, error) {
			requests++
			return "plain response", nil
		},
		onNoToolCalls: func(iteration int, response string) (turnLoopDirective, error) {
			return turnLoopBreak, nil
		},
		beforeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) {
			beforeCalled = true
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			executeCalled = true
			return turnLoopProceed, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopBreak {
		t.Fatalf("directive = %v, want %v", directive, turnLoopBreak)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want 1", requests)
	}
	if beforeCalled {
		t.Fatal("beforeToolCalls should not be called for no-tool break")
	}
	if executeCalled {
		t.Fatal("executeToolCalls should not be called for no-tool break")
	}
}

func TestRunTurnLoop_ExecuteToolCallsDoneDirectiveStopsLoop(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t, &commentSignalTool{})
	beforeCalled := false
	executeCalled := false

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 2,
		requestResponse: func(iteration int) (string, error) {
			return `{"tool":"comment_signal","args":{"note":"use search"}}`, nil
		},
		onNoToolCalls: func(iteration int, response string) (turnLoopDirective, error) {
			t.Fatal("onNoToolCalls should not be called when tool calls exist")
			return turnLoopProceed, nil
		},
		beforeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) {
			beforeCalled = true
			if len(toolCalls) != 1 {
				t.Fatalf("toolCalls len = %d, want 1", len(toolCalls))
			}
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			executeCalled = true
			if len(toolCalls) != 1 || toolCalls[0].Tool != "comment_signal" {
				t.Fatalf("unexpected toolCalls = %#v", toolCalls)
			}
			return turnLoopDone, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopDone {
		t.Fatalf("directive = %v, want %v", directive, turnLoopDone)
	}
	if !beforeCalled {
		t.Fatal("beforeToolCalls was not called")
	}
	if !executeCalled {
		t.Fatal("executeToolCalls was not called")
	}
}

func TestRunTurnLoop_AfterToolResultsRunsBeforeNextRequest(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t, &commentSignalTool{})
	events := []string{}

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 3,
		requestResponse: func(iteration int) (string, error) {
			events = append(events, "request")
			if iteration == 0 {
				return `{"tool":"comment_signal","args":{"note":"inspect"}}`, nil
			}
			return "plain response", nil
		},
		onNoToolCalls: func(iteration int, response string) (turnLoopDirective, error) {
			events = append(events, "no-tools")
			return turnLoopBreak, nil
		},
		beforeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) {
			events = append(events, "before-tools")
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			events = append(events, "execute-tools")
			return turnLoopProceed, nil
		},
		afterToolResults: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			events = append(events, "after-results")
			return turnLoopProceed, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopBreak {
		t.Fatalf("directive = %v, want %v", directive, turnLoopBreak)
	}
	want := []string{"request", "before-tools", "execute-tools", "after-results", "request", "no-tools"}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
}

func TestRunTurnLoop_AfterToolResultsBreakStopsNextRequest(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t, &commentSignalTool{})
	requests := 0

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 3,
		requestResponse: func(iteration int) (string, error) {
			requests++
			return `{"tool":"comment_signal","args":{"note":"inspect"}}`, nil
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopProceed, nil
		},
		afterToolResults: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopBreak, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopBreak {
		t.Fatalf("directive = %v, want %v", directive, turnLoopBreak)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want 1", requests)
	}
}

func TestRunTurnLoop_AfterToolResultsErrorPropagates(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t, &commentSignalTool{})
	expectedErr := errors.New("after results failed")

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 2,
		requestResponse: func(iteration int) (string, error) {
			return `{"tool":"comment_signal","args":{"note":"inspect"}}`, nil
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopProceed, nil
		},
		afterToolResults: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopProceed, expectedErr
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if directive != turnLoopReturn {
		t.Fatalf("directive = %v, want %v", directive, turnLoopReturn)
	}
}

func TestRunTurnLoop_AfterPrepareReturnDirectiveStopsImmediately(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t)
	requests := 0
	noToolCalls := 0

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 2,
		requestResponse: func(iteration int) (string, error) {
			requests++
			return "plain response", nil
		},
		afterPrepare: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopReturn, nil
		},
		onNoToolCalls: func(iteration int, response string) (turnLoopDirective, error) {
			noToolCalls++
			return turnLoopBreak, nil
		},
	})
	if err != nil {
		t.Fatalf("runTurnLoop() error = %v", err)
	}
	if directive != turnLoopReturn {
		t.Fatalf("directive = %v, want %v", directive, turnLoopReturn)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want 1", requests)
	}
	if noToolCalls != 0 {
		t.Fatalf("onNoToolCalls count = %d, want 0", noToolCalls)
	}
}

func TestRunTurnLoop_ExecuteToolCallsErrorPropagates(t *testing.T) {
	disableColors(t)

	runner := newTurnLoopTestRunner(t, &commentSignalTool{})
	expectedErr := errors.New("boom")

	directive, err := runner.runTurnLoop(turnLoopPolicy{
		hardLimit: 2,
		requestResponse: func(iteration int) (string, error) {
			return `{"tool":"comment_signal","args":{"note":"retry"}}`, nil
		},
		executeToolCalls: func(iteration int, response string, toolCalls []*tools.ToolCall) (turnLoopDirective, error) {
			return turnLoopProceed, expectedErr
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
	if directive != turnLoopReturn {
		t.Fatalf("directive = %v, want %v", directive, turnLoopReturn)
	}
}

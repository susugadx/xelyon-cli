package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"github.com/susugadx/xelyon-cli/internal/version"
)

func TestFailureHandlerConstructors(t *testing.T) {
	runner := &TurnRunner{}
	retry := &retryState{}
	step := &plan.PlanStep{ID: 3, Description: "apply fix"}
	state := &stepRunState{lastFailedResult: "boom", lastFailReason: "failed"}

	normal := newNormalModeFailureHandler(runner, retry, "boom")
	if normal.runner != runner || normal.retry != retry || normal.lastFailedResult != "boom" {
		t.Fatalf("newNormalModeFailureHandler() = %+v", normal)
	}

	p := &plan.Plan{Steps: []plan.PlanStep{*step}}
	planHandler := newPlanStepFailureHandler(runner, p, step, 0, retry, state)
	if planHandler.runner != runner || planHandler.plan != p || planHandler.step != step || planHandler.retry != retry || planHandler.state != state {
		t.Fatalf("newPlanStepFailureHandler() = %+v", planHandler)
	}
}

func TestFailureMessageBuilders(t *testing.T) {
	lastFailed := "tool failed"

	retryMsg := buildNormalModeRetryMessage(2, lastFailed)
	if !strings.Contains(retryMsg, "attempt 2") || !strings.Contains(retryMsg, lastFailed) {
		t.Fatalf("buildNormalModeRetryMessage() = %q", retryMsg)
	}

	strategyMsg := buildNormalModeStrategyChangeMessage(3, 4, lastFailed)
	if !strings.Contains(strategyMsg, "similar failure has now occurred 4 times") {
		t.Fatalf("buildNormalModeStrategyChangeMessage() = %q", strategyMsg)
	}

	planRetry := buildPlanStepRetryMessage(lastFailed, 1)
	if !strings.Contains(planRetry, lastFailed) || !strings.Contains(planRetry, "FAILED") {
		t.Fatalf("buildPlanStepRetryMessage() = %q", planRetry)
	}

	planStrategy := buildPlanStepStrategyChangeMessage(lastFailed, 3, 2)
	if !strings.Contains(planStrategy, "similar failure has now occurred 3 times") {
		t.Fatalf("buildPlanStepStrategyChangeMessage() = %q", planStrategy)
	}

	manual := buildPlanStepManualRetryMessage(lastFailed)
	if !strings.Contains(manual, "Do NOT skip this step") || !strings.Contains(manual, lastFailed) {
		t.Fatalf("buildPlanStepManualRetryMessage() = %q", manual)
	}

	comment := buildPlanStepCommentRetryMessage("please inspect logs", lastFailed)
	if !strings.Contains(comment, "please inspect logs") || !strings.Contains(comment, lastFailed) {
		t.Fatalf("buildPlanStepCommentRetryMessage() = %q", comment)
	}
}

func TestSpecialCommandRegistryAndHandlers(t *testing.T) {
	registry := specialCommandRegistry()
	for _, key := range []string{"/save", "/clear", "/version", "/plan", "/quit", "/q"} {
		if registry[key] == nil {
			t.Fatalf("specialCommandRegistry() missing %s", key)
		}
	}

	var out bytes.Buffer
	agent := &Agent{
		History: []api.Message{{Role: "user", Content: "hello"}},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleClearCommand(agent, nil) {
		t.Fatal("handleClearCommand() = false, want true")
	}
	if len(agent.History) != 0 {
		t.Fatalf("handleClearCommand() should clear history, got %+v", agent.History)
	}
	if !strings.Contains(out.String(), "History cleared") {
		t.Fatalf("handleClearCommand() output = %q", out.String())
	}

	out.Reset()
	if !handleVersionCommand(agent, nil) {
		t.Fatal("handleVersionCommand() = false, want true")
	}
	if !strings.Contains(out.String(), version.GetVersion()) {
		t.Fatalf("handleVersionCommand() output = %q, want version %q", out.String(), version.GetVersion())
	}
}

package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
)

func lastUserHistoryContent(history []api.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			return history[i].Content
		}
	}
	return ""
}

func TestNormalModeFailureHandler_HandleRetryFlows(t *testing.T) {
	disableColors(t)

	t.Run("first failure appends retry instruction", func(t *testing.T) {
		var out bytes.Buffer
		agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, newProjectMapDisabledConfig(), "", &out)
		runner := newTurnRunner(agent, context.Background())
		retry := &retryState{}

		if err := newNormalModeFailureHandler(runner, retry, "boom").Handle(); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if retry.count != 1 {
			t.Fatalf("retry.count = %d, want 1", retry.count)
		}
		if !strings.Contains(out.String(), "Failed (retry 1)") {
			t.Fatalf("Handle() output = %q, want retry log", out.String())
		}
		if got := lastUserHistoryContent(agent.History); !strings.Contains(got, "The previous tool execution FAILED") {
			t.Fatalf("last retry message = %q", got)
		}
	})

	t.Run("repeated failure escalates strategy change", func(t *testing.T) {
		var out bytes.Buffer
		agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, newProjectMapDisabledConfig(), "", &out)
		runner := newTurnRunner(agent, context.Background())
		retry := &retryState{
			count:       2,
			lastErrorFP: errorFingerprint("boom"),
			sameCount:   stalledRetryThreshold - 1,
		}

		if err := newNormalModeFailureHandler(runner, retry, "boom").Handle(); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if retry.sameCount != stalledRetryThreshold {
			t.Fatalf("retry.sameCount = %d, want %d", retry.sameCount, stalledRetryThreshold)
		}
		if !strings.Contains(out.String(), "Retrying with strategy change") {
			t.Fatalf("Handle() output = %q, want strategy-change log", out.String())
		}
		if got := lastUserHistoryContent(agent.History); !strings.Contains(got, "similar failure has now occurred") {
			t.Fatalf("last retry message = %q", got)
		}
	})

	t.Run("success after retries resets state", func(t *testing.T) {
		var out bytes.Buffer
		agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, newProjectMapDisabledConfig(), "", &out)
		runner := newTurnRunner(agent, context.Background())
		retry := &retryState{
			count:       2,
			lastErrorFP: errorFingerprint("boom"),
			sameCount:   2,
			stalledRuns: 1,
		}

		if err := newNormalModeFailureHandler(runner, retry, "").Handle(); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if retry.count != 0 || retry.sameCount != 0 || retry.stalledRuns != 0 {
			t.Fatalf("retry not reset: %+v", retry)
		}
		if !strings.Contains(out.String(), "Succeeded (on retry 2)") {
			t.Fatalf("Handle() output = %q, want success log", out.String())
		}
	})
}

func TestPlanStepFailureHandler_RetryFlows(t *testing.T) {
	disableColors(t)

	newStepPlan := func() (*plan.Plan, *plan.PlanStep) {
		p := &plan.Plan{
			Steps: []plan.PlanStep{{ID: 1, Description: "apply fix", Tools: []string{"read_file"}}},
		}
		return p, &p.Steps[0]
	}

	t.Run("first failure retries current step", func(t *testing.T) {
		var out bytes.Buffer
		provider := &sequenceMockProvider{name: "test", responses: []string{"done"}}
		agent := newTurnRunnerTestAgent(provider, newProjectMapDisabledConfig(), "", &out)
		runner := newTurnRunner(agent, context.Background())
		p, step := newStepPlan()
		retry := &retryState{}
		state := &stepRunState{lastFailedResult: "boom", lastFailReason: "failed"}

		handled, err := newPlanStepFailureHandler(runner, p, step, 0, retry, state).Handle()
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if !handled {
			t.Fatal("Handle() = false, want true")
		}
		if provider.callCount != 1 {
			t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
		}
		if !strings.Contains(out.String(), "Step 1 Failed (auto-retry 1)") {
			t.Fatalf("Handle() output = %q, want retry log", out.String())
		}
		if got := lastUserHistoryContent(agent.History); !strings.Contains(got, "The previous step FAILED") {
			t.Fatalf("last retry message = %q", got)
		}
	})

	t.Run("repeated failure retries with strategy change", func(t *testing.T) {
		var out bytes.Buffer
		provider := &sequenceMockProvider{name: "test", responses: []string{"done"}}
		agent := newTurnRunnerTestAgent(provider, newProjectMapDisabledConfig(), "", &out)
		runner := newTurnRunner(agent, context.Background())
		p, step := newStepPlan()
		retry := &retryState{
			count:       2,
			lastErrorFP: errorFingerprint("boom"),
			sameCount:   stalledRetryThreshold - 1,
		}
		state := &stepRunState{lastFailedResult: "boom", lastFailReason: "failed"}

		handled, err := newPlanStepFailureHandler(runner, p, step, 0, retry, state).Handle()
		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if !handled {
			t.Fatal("Handle() = false, want true")
		}
		if !strings.Contains(out.String(), "Retrying with strategy change") {
			t.Fatalf("Handle() output = %q, want strategy-change log", out.String())
		}
		if got := lastUserHistoryContent(agent.History); !strings.Contains(got, "similar failure has now occurred") {
			t.Fatalf("last retry message = %q", got)
		}
	})
}

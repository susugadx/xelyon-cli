package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	agentplan "github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newPlanImplementationRequestTestAgent() (*Agent, *bytes.Buffer) {
	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	return &Agent{Runtime: runtime}, &out
}

func TestNewPlanImplementationRequest(t *testing.T) {
	agent, _ := newPlanImplementationRequestTestAgent()
	ctx := context.Background()
	p := &agentplan.Plan{Summary: "ship it"}

	req := newPlanImplementationRequest(agent, ctx, "implement feature", p)
	if req.agent != agent || req.ctx != ctx || req.userRequest != "implement feature" || req.plan != p {
		t.Fatalf("newPlanImplementationRequest() = %#v, want captured fields", req)
	}
}

func TestPlanImplementationRequestRun_Success(t *testing.T) {
	disableColors(t)

	agent, out := newPlanImplementationRequestTestAgent()
	req := newPlanImplementationRequest(agent, context.Background(), "implement feature", &agentplan.Plan{})

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "All 0 steps completed!") {
		t.Fatalf("output = %q, want completion message", out.String())
	}
	if got := agent.statusRef().getStatus().State; got != StateWaitingInput {
		t.Fatalf("status = %q, want %q", got, StateWaitingInput)
	}
}

func TestPlanImplementationRequestRun_FailureSetsAbortedStatus(t *testing.T) {
	oldRun := runImplementationPhaseForPlanRequest
	t.Cleanup(func() { runImplementationPhaseForPlanRequest = oldRun })

	agent, _ := newPlanImplementationRequestTestAgent()
	runImplementationPhaseForPlanRequest = func(a *Agent, ctx context.Context, p *agentplan.Plan) error {
		return errors.New("boom")
	}

	req := newPlanImplementationRequest(agent, context.Background(), "implement feature", &agentplan.Plan{})
	err := req.Run()
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Run() error = %v, want boom", err)
	}
	if got := agent.statusRef().getStatus().State; got != StateAborted {
		t.Fatalf("status = %q, want %q", got, StateAborted)
	}
}

func TestPlanImplementationRequestHandleFailure_TokenLimitUsesRetry(t *testing.T) {
	oldHandle := handlePlanImplementationTokenLimit
	t.Cleanup(func() {
		handlePlanImplementationTokenLimit = oldHandle
	})

	agent, _ := newPlanImplementationRequestTestAgent()
	req := newPlanImplementationRequest(agent, context.Background(), "implement feature", &agentplan.Plan{})

	retryHookCalled := false
	handlePlanImplementationTokenLimit = func(a *Agent, err error, userRequest string, retryFunc func() error) bool {
		if !strings.Contains(err.Error(), "input tokens exceed") {
			t.Fatalf("unexpected error: %v", err)
		}
		if userRequest != "implement feature" {
			t.Fatalf("userRequest = %q, want %q", userRequest, "implement feature")
		}
		if retryFunc == nil {
			t.Fatal("retryFunc should be passed to token limit handler")
		}
		retryHookCalled = true
		return true
	}

	if err := req.handleFailure(errors.New("input tokens exceed model limit")); err != nil {
		t.Fatalf("handleFailure() error = %v, want nil", err)
	}
	if !retryHookCalled {
		t.Fatal("handleFailure() should delegate token limit retry handling")
	}
	if got := agent.statusRef().getStatus().State; got == StateAborted {
		t.Fatalf("status = %q, want non-aborted after handled token limit", got)
	}
}

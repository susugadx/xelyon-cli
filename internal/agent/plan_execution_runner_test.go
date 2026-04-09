package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestImplementationPhaseRunner_NextStepSkipsCompleted(t *testing.T) {
	agent := newPlanRequestTestAgent(t, &mockProvider{name: "test"}, "", &bytes.Buffer{})
	runner := newImplementationPhaseRunner(agent, context.Background(), &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Description: "done", Status: "completed"},
			{ID: 2, Description: "pending", Status: "pending"},
		},
	})

	stepID, step, ok := runner.nextStep()
	if !ok {
		t.Fatal("nextStep() ok = false, want true")
	}
	if stepID != 2 {
		t.Fatalf("nextStep() stepID = %d, want 2", stepID)
	}
	if step == nil || step.Description != "pending" {
		t.Fatalf("nextStep() step = %#v", step)
	}
}

func TestImplementationPhaseRunner_Run_CompletesPlanAndRunsHooks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedStepProvider{response: "Plan step final response"}
	agent := newPlanRequestTestAgent(t, provider, "", &out)
	agent.cfg().Hooks.OnStepComplete = []string{"echo ok"}
	agent.cfg().Hooks.Timeout = 10
	p := &plan.Plan{
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Finish the step",
			Status:      "pending",
		}},
	}

	if err := newImplementationPhaseRunner(agent, context.Background(), p).Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := p.GetStep(1).Status; got != "completed" {
		t.Fatalf("step status = %q, want completed", got)
	}
	if !strings.Contains(out.String(), "Running step complete hook") {
		t.Fatalf("expected step hook output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "All 1 steps completed") {
		t.Fatalf("expected completion output, got %q", out.String())
	}
}

func TestImplementationPhaseRunner_Run_PropagatesStepError(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newPlanRequestTestAgent(t, &mockErrorProvider{}, "", &out)
	p := &plan.Plan{
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Fail the step",
			Status:      "pending",
		}},
	}

	err := newImplementationPhaseRunner(agent, context.Background(), p).Run()
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "step 1 failed: mock error") {
		t.Fatalf("Run() error = %v", err)
	}
	if got := p.GetStep(1).Status; got != "pending" {
		t.Fatalf("step status = %q, want pending", got)
	}
}

func TestImplementationPhaseRunner_PersistSessionSavesMetadata(t *testing.T) {
	disableColors(t)

	agent := newPlanRequestTestAgent(t, &mockProvider{name: "test"}, "", &bytes.Buffer{})
	runner := newImplementationPhaseRunner(agent, context.Background(), &plan.Plan{})

	if err := runner.persistSession(); err != nil {
		t.Fatalf("persistSession() error = %v", err)
	}

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected saved session metadata")
	}

	found := false
	for _, session := range sessions {
		if session.ID == agent.session.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("saved sessions do not contain current session id %q", agent.session.ID)
	}
}

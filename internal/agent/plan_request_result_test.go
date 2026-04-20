package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestPlanModeRequest_HandleInvestigationResult_NoImplementationCases(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name    string
		plan    *plan.Plan
		message string
	}{
		{name: "nil plan", plan: nil, message: "No implementation needed."},
		{name: "empty plan", plan: &plan.Plan{Steps: []plan.PlanStep{}}, message: "No implementation steps needed."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			agent := newPlanRequestTestAgent(t, &mockProvider{name: "test"}, "", &out)
			req := newPlanModeRequest(agent, context.Background(), "investigate only")

			handled, err := req.handleInvestigationResult(tt.plan)
			if err != nil {
				t.Fatalf("handleInvestigationResult() error = %v", err)
			}
			if !handled {
				t.Fatal("handleInvestigationResult() handled = false, want true")
			}
			if !strings.Contains(out.String(), tt.message) {
				t.Fatalf("expected output to contain %q, got %q", tt.message, out.String())
			}
			if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
				t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
			}
		})
	}
}

func TestPlanModeRequest_HandleInvestigationResult_PlanApprovalStopsAfterApproval(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newPlanRequestTestAgent(t, &mockProvider{name: "test"}, "1\n", &out)
	agent.PlanModeEnabled = true
	req := newPlanModeRequest(agent, context.Background(), "implement feature")
	p := &plan.Plan{
		Summary: "Ship a small change",
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update the target file",
			Tools:       []string{"read_file", "str_replace"},
		}},
	}

	handled, err := req.handleInvestigationResult(p)
	if err != nil {
		t.Fatalf("handleInvestigationResult() error = %v", err)
	}
	if !handled {
		t.Fatal("handleInvestigationResult() handled = false, want true")
	}
	if !strings.Contains(out.String(), "Implementation Plan") {
		t.Fatalf("expected rendered plan in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Plan approved. Plan Mode complete. Implementation not started.") {
		t.Fatalf("expected approval output, got %q", out.String())
	}
	if agent.PlanModeEnabled {
		t.Fatal("PlanModeEnabled should be false after approval")
	}
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestPlanModeRequest_Run_PlanApprovalStopsBeforeImplementation(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Ship a small change\",\"steps\":[{\"id\":1,\"description\":\"Update the target file\",\"tools\":[\"str_replace\"]}]}}\n```", nil
			default:
				return "unexpected extra provider call", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true
	req := newPlanModeRequest(agent, context.Background(), "implement feature")

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	output := out.String()
	if !strings.Contains(output, "Implementation Plan") {
		t.Fatalf("expected rendered plan in output, got %q", output)
	}
	if !strings.Contains(output, "Plan approved. Plan Mode complete. Implementation not started.") {
		t.Fatalf("expected approval output, got %q", output)
	}
	if strings.Contains(output, "Starting implementation") {
		t.Fatalf("expected no implementation start message, got %q", output)
	}
	if agent.PlanModeEnabled {
		t.Fatal("PlanModeEnabled should be false after approval")
	}
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

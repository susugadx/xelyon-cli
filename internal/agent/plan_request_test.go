package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newPlanRequestTestAgent(t *testing.T, provider api.Provider, input string, out *bytes.Buffer) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	runtime := NewAgentRuntimeWithConfig(newChatRequestTestConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(input), out, out)
	runtime.Registry = tools.DefaultRegistry.Clone()

	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	agent.setAutoApprove(true)
	return agent
}

func TestPlanModeRequest_HandleInvestigationResult_NoImplementationCases(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name    string
		plan    *plan.Plan
		message string
	}{
		{
			name:    "nil plan",
			plan:    nil,
			message: "No implementation needed.",
		},
		{
			name:    "empty plan",
			plan:    &plan.Plan{Steps: []plan.PlanStep{}},
			message: "No implementation steps needed.",
		},
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

func TestPlanModeRequest_HandleInvestigationResult_ApprovedPlanContinues(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newPlanRequestTestAgent(t, &mockProvider{name: "test"}, "1\n", &out)
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
	if handled {
		t.Fatal("handleInvestigationResult() handled = true, want false")
	}
	if !strings.Contains(out.String(), "Implementation Plan") {
		t.Fatalf("expected rendered plan in output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Plan approved. Starting implementation") {
		t.Fatalf("expected approval output, got %q", out.String())
	}
	if status := agent.statusRef().getStatus(); status.State != StateRunning {
		t.Fatalf("status.State = %q, want %q", status.State, StateRunning)
	}
}

func TestPlanModeRequest_HandleInvestigationResult_FeedbackRestartsPlanMode(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: []string{"Investigation complete."},
	}
	agent := newPlanRequestTestAgent(t, provider, "3\nNeed more evidence\n\n", &out)
	req := newPlanModeRequest(agent, context.Background(), "implement feature")
	p := &plan.Plan{
		Summary: "Ship a small change",
		Steps: []plan.PlanStep{{
			ID:          1,
			Description: "Update the target file",
		}},
	}

	handled, err := req.handleInvestigationResult(p)
	if err != nil {
		t.Fatalf("handleInvestigationResult() error = %v", err)
	}
	if !handled {
		t.Fatal("handleInvestigationResult() handled = false, want true")
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if !strings.Contains(out.String(), "Plan rejected with feedback: Need more evidence") {
		t.Fatalf("expected feedback output, got %q", out.String())
	}
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestPlanModeRequest_Run_TokenLimitRetrySuccess(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "openai",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "", errors.New("input tokens exceed model limit")
			case 1:
				return "compressed summary", nil
			default:
				return "Investigation complete.", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "", &out)
	seedHistoryForTokenRetry(agent, 6)
	req := newPlanModeRequest(agent, context.Background(), "investigate only")

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}
	if !strings.Contains(out.String(), "✅ 自動圧縮＆リトライ成功") {
		t.Fatalf("expected retry success output, got %q", out.String())
	}
	if agent.tokenLimitRetryCount != 0 {
		t.Fatalf("tokenLimitRetryCount = %d, want 0", agent.tokenLimitRetryCount)
	}
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestPlanModeRequest_Run_PropagatesContextCanceled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &blockingCancelProvider{started: make(chan struct{})}
	agent := newPlanRequestTestAgent(t, provider, "", &out)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- newPlanModeRequest(agent, ctx, "investigate only").Run()
	}()

	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("ChatWithTools was not called")
	}
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

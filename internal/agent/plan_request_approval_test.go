package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

type planApprovalContextKey struct{}

type planApprovalContextPrompter struct {
	ctx context.Context
	req ui.PromptRequest
}

func (p *planApprovalContextPrompter) Prompt(ctx context.Context, req ui.PromptRequest) (ui.PromptResponse, error) {
	p.ctx = ctx
	p.req = req
	return ui.PromptResponse{Action: ui.PromptActionNo}, nil
}

type planApprovalBlockingPrompter struct {
	started chan struct{}
}

func (p *planApprovalBlockingPrompter) Prompt(ctx context.Context, _ ui.PromptRequest) (ui.PromptResponse, error) {
	close(p.started)
	select {
	case <-ctx.Done():
		return ui.PromptResponse{Action: ui.PromptActionNo, Cancelled: true}, nil
	case <-time.After(200 * time.Millisecond):
		return ui.PromptResponse{Action: ui.PromptActionYes}, nil
	}
}

func TestConfirmPlan_UsesRuntimePromptIO(t *testing.T) {
	var out bytes.Buffer

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("2\n"), &out, &out),
		},
	}

	approved, feedback := agent.confirmPlan(context.Background())
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if !strings.Contains(out.String(), "Approve this plan?") {
		t.Fatalf("expected runtime output to contain confirmation prompt, got %q", out.String())
	}
}

func TestConfirmPlan_UsesRequestContextForPrompt(t *testing.T) {
	var out bytes.Buffer
	prompter := &planApprovalContextPrompter{}
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &out)
	runtime.SetPrompter(prompter)
	ctx := context.WithValue(context.Background(), planApprovalContextKey{}, "plan-request")

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: runtime,
		},
	}

	approved, feedback := agent.confirmPlan(ctx)
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if prompter.ctx == nil {
		t.Fatal("prompter did not receive context")
	}
	if got := prompter.ctx.Value(planApprovalContextKey{}); got != "plan-request" {
		t.Fatalf("prompt context marker = %v, want plan-request", got)
	}
	if prompter.req.Kind != ui.PromptKindConfirm || prompter.req.Message != "Approve this plan?" || !prompter.req.AllowComment {
		t.Fatalf("prompt request = %#v, want plan approval confirm", prompter.req)
	}
}

func TestConfirmPlan_RequestContextCancellationCancelsPrompt(t *testing.T) {
	var out bytes.Buffer
	prompter := &planApprovalBlockingPrompter{started: make(chan struct{})}
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &out)
	runtime.SetPrompter(prompter)
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: runtime,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan struct {
		approved bool
		feedback string
	}, 1)

	go func() {
		approved, feedback := agent.confirmPlan(ctx)
		resultCh <- struct {
			approved bool
			feedback string
		}{approved: approved, feedback: feedback}
	}()

	select {
	case <-prompter.started:
	case <-time.After(time.Second):
		t.Fatal("prompt did not start")
	}
	cancel()

	select {
	case result := <-resultCh:
		if result.approved {
			t.Fatal("confirmPlan() approved = true after context cancellation, want false")
		}
		if result.feedback != "" {
			t.Fatalf("confirmPlan() feedback = %q after context cancellation, want empty", result.feedback)
		}
	case <-time.After(time.Second):
		t.Fatal("confirmPlan() did not return after request context cancellation")
	}
}

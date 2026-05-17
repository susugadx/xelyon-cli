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

type planApprovalResponsePrompter struct {
	resp ui.PromptResponse
}

func (p planApprovalResponsePrompter) Prompt(ctx context.Context, req ui.PromptRequest) (ui.PromptResponse, error) {
	return p.resp, nil
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

func TestConfirmPlan_UsesPlanFeedbackPromptResponse(t *testing.T) {
	var out bytes.Buffer
	runtime := ui.NewRuntime(strings.NewReader(""), &out, &out)
	runtime.SetPrompter(planApprovalResponsePrompter{
		resp: ui.PromptResponse{Action: ui.PromptActionComment, Text: "  add tests first  "},
	})
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: runtime,
		},
	}

	approved, feedback := agent.confirmPlan(context.Background())
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "add tests first" {
		t.Fatalf("confirmPlan() feedback = %q, want trimmed feedback", feedback)
	}
}

func TestConfirmPlan_UsesRuntimePromptIO(t *testing.T) {
	var out bytes.Buffer

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("n\n"), &out, &out),
		},
	}

	approved, feedback := agent.confirmPlan(context.Background())
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if !strings.Contains(out.String(), "Approve the plan, request changes, or cancel Plan Mode.") {
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
	if prompter.req.Kind != ui.PromptKindConfirm ||
		prompter.req.Title != "Review implementation plan" ||
		prompter.req.Message != "Approve the plan, request changes, or cancel Plan Mode." ||
		!prompter.req.AllowComment ||
		prompter.req.ConfirmSubmitPolicy != ui.PromptConfirmSubmitExplicit {
		t.Fatalf("prompt request = %#v, want plan approval confirm", prompter.req)
	}
	if len(prompter.req.Options) != 3 ||
		prompter.req.Options[0].Value != string(ui.PromptActionYes) ||
		prompter.req.Options[1].Value != string(ui.PromptActionComment) ||
		prompter.req.Options[2].Value != string(ui.PromptActionNo) {
		t.Fatalf("prompt options = %#v, want approve/comment/cancel actions", prompter.req.Options)
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

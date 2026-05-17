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
)

func TestPlanModeRequest_HandleInvestigationResult_FeedbackRestartsPlanMode(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: []string{"Investigation complete."},
	}
	agent := newPlanRequestTestAgent(t, provider, "c\nNeed more evidence\n\n", &out)
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
	if !strings.Contains(out.String(), "Plan feedback received. Regenerating plan: Need more evidence") {
		t.Fatalf("expected feedback output, got %q", out.String())
	}
	if req.handoff != nil {
		t.Fatal("feedback should not create an implementation handoff")
	}
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestPlanModeRequest_Run_FeedbackRerunRecordsUserRequestInSession(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Task B\",\"steps\":[{\"id\":1,\"description\":\"Implement task B\",\"tools\":[\"str_replace\"]}]}}\n```", nil
			case 1:
				return "Investigation complete.", nil
			default:
				return "unexpected extra provider call", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "c\nNeed more evidence\n\n", &out)
	req := newPlanModeRequest(agent, context.Background(), "task b")

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 2 {
		t.Fatalf("len(agent.session.Messages) = %d, want 2 after feedback rerun", len(agent.session.Messages))
	}
	if agent.session.Messages[0].Role != "user" {
		t.Fatalf("session.Messages[0].Role = %q, want user", agent.session.Messages[0].Role)
	}
	if !strings.Contains(agent.session.Messages[0].Content, "Previous plan feedback: Need more evidence") {
		t.Fatalf("session.Messages[0].Content = %q, want rerun request with feedback", agent.session.Messages[0].Content)
	}
	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2 after feedback rerun", len(loadedMessages))
	}
	if loadedMessages[0].Role != "user" {
		t.Fatalf("loaded.ToAPIMessages()[0].Role = %q, want user", loadedMessages[0].Role)
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

func TestPlanModeRequest_Run_TokenLimitRetryRecordsUserRequestInSession(t *testing.T) {
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
			case 2:
				return "Investigation complete.", nil
			default:
				return "unexpected extra provider call", nil
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
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 2 {
		t.Fatalf("len(agent.session.Messages) = %d, want 2 after token-limit rerun", len(agent.session.Messages))
	}
	if agent.session.Messages[0].Role != "user" {
		t.Fatalf("session.Messages[0].Role = %q, want user", agent.session.Messages[0].Role)
	}
	if strings.TrimSpace(agent.session.Messages[0].Content) != "investigate only" {
		t.Fatalf("session.Messages[0].Content = %q, want original rerun request", agent.session.Messages[0].Content)
	}
	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2 after token-limit rerun", len(loadedMessages))
	}
	if loadedMessages[0].Role != "user" {
		t.Fatalf("loaded.ToAPIMessages()[0].Role = %q, want user", loadedMessages[0].Role)
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

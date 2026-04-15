package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestPlanModeRequest_Run_TechnicalFailureRestoresPreviousApprovedPlan(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	previousPlan := "Implementation Plan\n1. Implement task A"
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "", errors.New("temporary api error")
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "", &out)
	agent.setPendingApprovedPlanState(previousPlan, true, []string{"internal/agent/task_a.go"})
	req := newPlanModeRequest(agent, context.Background(), "task b")

	err := req.Run()
	if err == nil || !strings.Contains(err.Error(), "temporary api error") {
		t.Fatalf("Run() error = %v, want temporary api error", err)
	}
	if agent.PendingApprovedPlan != previousPlan {
		t.Fatalf("PendingApprovedPlan = %q, want restored previous plan %q", agent.PendingApprovedPlan, previousPlan)
	}
	if agent.session == nil || agent.session.PendingApprovedPlan != previousPlan {
		t.Fatalf("session.PendingApprovedPlan = %q, want restored previous plan %q", agent.session.PendingApprovedPlan, previousPlan)
	}
	if !agent.PendingApprovedPlanHasChanges {
		t.Fatal("PendingApprovedPlanHasChanges should be restored after technical failure")
	}
	if agent.session == nil || !agent.session.PendingApprovedPlanHasChanges {
		t.Fatal("session.PendingApprovedPlanHasChanges should be restored after technical failure")
	}

	normalReq := &chatRequest{input: "resume previous approved plan"}
	if err := agent.executeChatRequest(context.Background(), normalReq); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if normalReq.approvedPlanHandoff != previousPlan {
		t.Fatalf("approvedPlanHandoff = %q, want restored previous plan %q", normalReq.approvedPlanHandoff, previousPlan)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after follow-up completion", agent.PendingApprovedPlan)
	}
	if agent.PendingApprovedPlanHasChanges {
		t.Fatal("PendingApprovedPlanHasChanges = true, want false after follow-up completion")
	}
}

func TestPlanModeRequest_Run_TokenLimitRetryFailureRestoresPreviousApprovedPlan(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	previousPlan := "Implementation Plan\n1. Implement task A"
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
				return "", errors.New("temporary retry api error")
			case 3:
				return approvedPlanWriteCall("internal/agent/restored_token_retry_plan.go"), nil
			default:
				return "The requested changes are done.", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "", &out)
	registerApprovedPlanWriteTool(agent)
	seedHistoryForTokenRetry(agent, 6)
	agent.setPendingApprovedPlan(previousPlan)
	req := newPlanModeRequest(agent, context.Background(), "task b")

	err := req.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want token-limit retry failure")
	}
	if agent.PendingApprovedPlan != previousPlan {
		t.Fatalf("PendingApprovedPlan = %q, want restored previous plan %q", agent.PendingApprovedPlan, previousPlan)
	}
	if agent.session == nil || agent.session.PendingApprovedPlan != previousPlan {
		t.Fatalf("session.PendingApprovedPlan = %q, want restored previous plan %q", agent.session.PendingApprovedPlan, previousPlan)
	}

	normalReq := &chatRequest{input: "resume previous approved plan"}
	if err := agent.executeChatRequest(context.Background(), normalReq); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if normalReq.approvedPlanHandoff != previousPlan {
		t.Fatalf("approvedPlanHandoff = %q, want restored previous plan %q", normalReq.approvedPlanHandoff, previousPlan)
	}
}

func TestPlanModeRequest_Run_CancelClearsStalePendingPlanAndPreventsHandoff(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "Here is the new plan:\n```json\n{\"plan\":{\"summary\":\"Task B\",\"steps\":[{\"id\":1,\"description\":\"Implement task B\",\"tools\":[\"str_replace\"]}]}}\n```", nil
			default:
				snapshot := append([]api.Message(nil), history...)
				capturedHistories = append(capturedHistories, snapshot)
				return "done", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "2\n", &out)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Implement task A"
	req := newPlanModeRequest(agent, context.Background(), "task b")

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if agent.PendingApprovedPlan != "" {
		t.Fatalf("PendingApprovedPlan = %q, want empty after new /plan cancel", agent.PendingApprovedPlan)
	}

	normalReq := &chatRequest{input: "do something unrelated"}
	if err := agent.executeChatRequest(context.Background(), normalReq); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if normalReq.approvedPlanHandoff != "" {
		t.Fatalf("approvedPlanHandoff = %q, want empty after cancelled /plan", normalReq.approvedPlanHandoff)
	}
	if len(capturedHistories) != 1 {
		t.Fatalf("capturedHistories = %d, want 1", len(capturedHistories))
	}
	lastUserMessage := capturedHistories[0][len(capturedHistories[0])-1].Content
	if strings.Contains(lastUserMessage, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("expected no handoff guidance after cancelled /plan, got %q", lastUserMessage)
	}
	if strings.Contains(lastUserMessage, "task A") {
		t.Fatalf("expected stale plan not to leak after cancelled /plan, got %q", lastUserMessage)
	}
}

func TestPlanModeRequest_Run_ApproveReplacesStalePendingPlanForNextHandoff(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			switch call {
			case 0:
				return "Here is the new plan:\n```json\n{\"plan\":{\"summary\":\"Task B\",\"steps\":[{\"id\":1,\"description\":\"Implement task B\",\"tools\":[\"str_replace\"]}]}}\n```", nil
			default:
				snapshot := append([]api.Message(nil), history...)
				capturedHistories = append(capturedHistories, snapshot)
				return "done", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PendingApprovedPlan = "Implementation Plan\n1. Implement task A"
	req := newPlanModeRequest(agent, context.Background(), "task b")

	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(agent.PendingApprovedPlan, "Implement task B") {
		t.Fatalf("PendingApprovedPlan = %q, want task B plan", agent.PendingApprovedPlan)
	}
	if strings.Contains(agent.PendingApprovedPlan, "task A") {
		t.Fatalf("PendingApprovedPlan = %q, stale task A plan should be replaced", agent.PendingApprovedPlan)
	}

	normalReq := &chatRequest{input: "implement approved plan"}
	if err := agent.executeChatRequest(context.Background(), normalReq); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if !strings.Contains(normalReq.approvedPlanHandoff, "Implement task B") {
		t.Fatalf("approvedPlanHandoff = %q, want task B plan", normalReq.approvedPlanHandoff)
	}
	if strings.Contains(normalReq.approvedPlanHandoff, "task A") {
		t.Fatalf("approvedPlanHandoff = %q, stale task A plan should not leak", normalReq.approvedPlanHandoff)
	}
	if len(capturedHistories) != 1 {
		t.Fatalf("capturedHistories = %d, want 1", len(capturedHistories))
	}
	lastUserMessage := capturedHistories[0][len(capturedHistories[0])-1].Content
	if !strings.Contains(lastUserMessage, "Implement task B") {
		t.Fatalf("expected task B handoff guidance, got %q", lastUserMessage)
	}
	if strings.Contains(lastUserMessage, "task A") {
		t.Fatalf("expected stale plan not to leak into handoff, got %q", lastUserMessage)
	}
}

func TestPlanModeRequest_Run_ApprovalRestoresConversationBeforeNormalMode(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			switch call {
			case 0:
				return "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Task B\",\"steps\":[{\"id\":1,\"description\":\"Implement task B\",\"tools\":[\"str_replace\"]}]}}\n```", nil
			default:
				return "done", nil
			}
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.History = []api.Message{{Role: "user", Content: "previous normal conversation"}}
	if agent.session != nil {
		agent.session.AddMessage("user", "previous normal conversation", agent.CurrentModel)
	}

	req := newPlanModeRequest(agent, context.Background(), "task b")
	if err := req.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(agent.History) != 1 {
		t.Fatalf("len(agent.History) = %d, want 1 after plan-mode rollback", len(agent.History))
	}
	if got := agent.History[0].Content; got != "previous normal conversation" {
		t.Fatalf("agent.History[0].Content = %q, want previous conversation", got)
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 1 {
		t.Fatalf("len(agent.session.Messages) = %d, want 1 after plan-mode rollback", len(agent.session.Messages))
	}

	normalReq := &chatRequest{input: "implement approved plan"}
	if err := agent.executeChatRequest(context.Background(), normalReq); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if len(capturedHistories) != 2 {
		t.Fatalf("capturedHistories = %d, want 2", len(capturedHistories))
	}
	normalHistory := capturedHistories[1]
	for _, msg := range normalHistory {
		if strings.Contains(msg.Content, "READ-ONLY ONLY") {
			t.Fatalf("normal mode history should not retain investigation prompt, got %#v", normalHistory)
		}
		if strings.Contains(msg.Content, "Modification tools are FORBIDDEN") {
			t.Fatalf("normal mode history should not retain read-only restriction, got %#v", normalHistory)
		}
	}
}

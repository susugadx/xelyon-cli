package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestChatCore_PlanModeEnabledUsesPlanModeFlow(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: []string{"Investigation complete."},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("investigate only", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if len(agent.History) == 0 {
		t.Fatal("History should retain investigation-only conversation")
	}
	if got := agent.History[len(agent.History)-1].Content; !strings.Contains(got, "Investigation complete.") {
		t.Fatalf("last history message = %q, want investigation result", got)
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 2 {
		t.Fatalf("len(agent.session.Messages) = %d, want 2 for user request + investigation result", len(agent.session.Messages))
	}
	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2 after investigation-only /plan", len(loadedMessages))
	}
	if got := loadedMessages[len(loadedMessages)-1].Content; !strings.Contains(got, "Investigation complete.") {
		t.Fatalf("loaded last message = %q, want investigation result", got)
	}
	if strings.Contains(agent.SystemPrompt, "You are in Plan Mode - producing a text plan") {
		t.Fatalf("planning prompt should not remain in SystemPrompt after no-op /plan: %q", agent.SystemPrompt)
	}
	if !strings.Contains(out.String(), "Investigation phase - researching the codebase") {
		t.Fatalf("expected plan mode output, got %q", out.String())
	}
	status := agent.statusRef().getStatus()
	if status.State != StateWaitingInput {
		t.Fatalf("status.State = %q, want %q", status.State, StateWaitingInput)
	}
}

func TestChatCore_PlanModeApprovalRestoresSystemPromptAndRemovesPlanningRequestFromSession(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			if call != 0 {
				return "unexpected extra provider call", nil
			}
			return "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Ship a small change\",\"steps\":[{\"id\":1,\"description\":\"Update foo.go and tests\",\"tools\":[\"str_replace\"]}]}}\n```", nil
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true
	req := &chatRequest{input: "implement feature"}
	agent.prepareChatRequest(req)
	baselineSystemPrompt := agent.SystemPrompt
	ctx, cleanup := agent.beginChatRequestContext()
	defer cleanup()

	if err := agent.executeChatRequest(ctx, req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if agent.SystemPrompt != baselineSystemPrompt {
		t.Fatalf("SystemPrompt should be restored after /plan approval")
	}
	if strings.Contains(agent.SystemPrompt, "You are in Plan Mode - producing a text plan") {
		t.Fatalf("planning prompt should not remain in SystemPrompt: %q", agent.SystemPrompt)
	}
	if len(agent.History) != 0 {
		t.Fatalf("len(agent.History) = %d, want 0 after /plan approval rollback", len(agent.History))
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 0 {
		t.Fatalf("len(agent.session.Messages) = %d, want 0 after /plan approval rollback", len(agent.session.Messages))
	}
	if agent.PendingApprovedPlan == "" {
		t.Fatal("PendingApprovedPlan should remain after approval")
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if len(loaded.ToAPIMessages()) != 0 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 0 after saving approved handoff session", len(loaded.ToAPIMessages()))
	}
	if loaded.PendingApprovedPlan == "" {
		t.Fatal("loaded.PendingApprovedPlan should be preserved across save/load")
	}
	if strings.Contains(loaded.PendingApprovedPlan, "implement feature") {
		t.Fatalf("loaded.PendingApprovedPlan should contain only plan text, got %q", loaded.PendingApprovedPlan)
	}
}

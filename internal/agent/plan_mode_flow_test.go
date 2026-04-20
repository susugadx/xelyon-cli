package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type planResponseContextProvider struct {
	callCount               int
	responses               []string
	responseID              string
	usedPreviousResponseIDs []string
	histories               [][]api.Message
}

func (p *planResponseContextProvider) Name() string { return "openai" }

func (p *planResponseContextProvider) SupportsImages() bool { return false }

func (p *planResponseContextProvider) IsFunctionCallingEnabled() bool { return true }

func (p *planResponseContextProvider) HasCachedResponseID() bool {
	return p.responseID != ""
}

func (p *planResponseContextProvider) SetResponseID(id string) {
	p.responseID = id
}

func (p *planResponseContextProvider) GetResponseID() string {
	return p.responseID
}

func (p *planResponseContextProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.usedPreviousResponseIDs = append(p.usedPreviousResponseIDs, p.responseID)
	snapshot := append([]api.Message(nil), history...)
	p.histories = append(p.histories, snapshot)

	call := p.callCount
	p.callCount++

	response := "done"
	if len(p.responses) > 0 {
		if call < len(p.responses) {
			response = p.responses[call]
		} else {
			response = p.responses[len(p.responses)-1]
		}
	}

	p.responseID = fmt.Sprintf("resp_plan_chain_%d", call)
	return response, nil
}

func (p *planResponseContextProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

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

func TestChatCore_PlanModeApproval_EndsPlanModeWithoutStartingImplementation(t *testing.T) {
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
		t.Fatalf("len(agent.History) = %d, want 0 after plan-only approval rollback", len(agent.History))
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 0 {
		t.Fatalf("len(agent.session.Messages) = %d, want 0 after plan-only approval rollback", len(agent.session.Messages))
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if len(loaded.ToAPIMessages()) != 0 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 0 after plan-only approval rollback", len(loaded.ToAPIMessages()))
	}
	if strings.Contains(out.String(), "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("plan approval should not create legacy marker, got %q", out.String())
	}
}

func TestChatCore_PlanModeApproval_DoesNotCarryPlanningResponseContextToNextNormalTurnOnNewSession(t *testing.T) {
	disableColors(t)

	planResponse := "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Ship a small change\",\"steps\":[{\"id\":1,\"description\":\"Update foo.go and tests\",\"tools\":[\"str_replace\"]}]}}\n```"

	var out bytes.Buffer
	provider := &planResponseContextProvider{
		responses: []string{
			planResponse,
			"The requested changes are done.",
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("first chatCore() error = %v", err)
	}
	if provider.GetResponseID() != "" {
		t.Fatalf("provider responseID after plan approval = %q, want empty (planning chain must be detached)", provider.GetResponseID())
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID after plan approval = %q, want empty", agent.session.ResponseID)
	}
	if len(agent.History) != 0 {
		t.Fatalf("len(agent.History) after plan approval = %d, want 0 (investigation history must be rolled back)", len(agent.History))
	}
	if len(agent.session.Messages) != 0 {
		t.Fatalf("len(agent.session.Messages) after plan approval = %d, want 0 (investigation history must be rolled back)", len(agent.session.Messages))
	}

	if err := agent.chatCore("continue", nil, false); err != nil {
		t.Fatalf("second chatCore() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if len(provider.usedPreviousResponseIDs) != 2 {
		t.Fatalf("len(usedPreviousResponseIDs) = %d, want 2", len(provider.usedPreviousResponseIDs))
	}
	if provider.usedPreviousResponseIDs[1] != "" {
		t.Fatalf("second turn previous_response_id = %q, want empty to avoid planning-chain continuation", provider.usedPreviousResponseIDs[1])
	}
	if len(provider.histories) != 2 {
		t.Fatalf("len(provider.histories) = %d, want 2", len(provider.histories))
	}
	for _, msg := range provider.histories[1] {
		if strings.Contains(msg.Content, "READ-ONLY ONLY") {
			t.Fatalf("next normal turn history must not contain investigation restriction, got %q", msg.Content)
		}
		if strings.Contains(msg.Content, "Modification tools are FORBIDDEN") {
			t.Fatalf("next normal turn history must not contain investigation restriction, got %q", msg.Content)
		}
		if strings.Contains(msg.Content, "You are in PLAN MODE (Investigation Phase).") {
			t.Fatalf("next normal turn history must not contain investigation prompt, got %q", msg.Content)
		}
	}
}

func TestChatCore_PlanModeApproval_ClearsPrePlanResponseContextForNextNormalTurn(t *testing.T) {
	disableColors(t)

	planResponse := "Here is the plan:\n```json\n{\"plan\":{\"summary\":\"Ship a small change\",\"steps\":[{\"id\":1,\"description\":\"Update foo.go and tests\",\"tools\":[\"str_replace\"]}]}}\n```"

	var out bytes.Buffer
	provider := &planResponseContextProvider{
		responseID: "resp_before_plan",
		responses: []string{
			planResponse,
			"The requested changes are done.",
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("first chatCore() error = %v", err)
	}
	if provider.GetResponseID() != "" {
		t.Fatalf("provider responseID after plan approval = %q, want empty", provider.GetResponseID())
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID after plan approval = %q, want empty", agent.session.ResponseID)
	}

	if err := agent.chatCore("continue", nil, false); err != nil {
		t.Fatalf("second chatCore() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if len(provider.usedPreviousResponseIDs) != 2 {
		t.Fatalf("len(usedPreviousResponseIDs) = %d, want 2", len(provider.usedPreviousResponseIDs))
	}
	if provider.usedPreviousResponseIDs[1] != "" {
		t.Fatalf("second turn previous_response_id = %q, want empty to force history-based normal turn", provider.usedPreviousResponseIDs[1])
	}
}

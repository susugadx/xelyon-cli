package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	planModeFlowStructuredRelatedFilesFragment = "Related files: foo.go, foo_test.go"
	planModeFlowImplementationResponse         = "The requested changes are done."
	planModeFlowImplementationStartText        = "Starting implementation from approved plan"
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

func newPlanApprovalFlowProvider(initialResponseID string) *planResponseContextProvider {
	return &planResponseContextProvider{
		responseID: initialResponseID,
		responses: []string{
			planHandoffTestApprovedPlanResponse(planHandoffTestApprovedStep),
			planModeFlowImplementationResponse,
		},
	}
}

func assertPlanAndImplementationProviderCalls(t *testing.T, provider *planResponseContextProvider) {
	t.Helper()
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2 for planning + implementation", provider.callCount)
	}
	if len(provider.histories) != 2 {
		t.Fatalf("len(provider.histories) = %d, want 2", len(provider.histories))
	}
}

func planImplementationHistory(t *testing.T, provider *planResponseContextProvider) []api.Message {
	t.Helper()
	assertPlanAndImplementationProviderCalls(t, provider)
	return provider.histories[1]
}

func assertImplementationHistoryContainsApprovedPlanHandoff(t *testing.T, history []api.Message) {
	t.Helper()
	if len(history) == 0 {
		t.Fatal("implementation history should contain handoff input")
	}
	handoffContent := history[len(history)-1].Content
	if !strings.Contains(handoffContent, "Implement the approved plan now.") || !strings.Contains(handoffContent, planHandoffTestApprovedStep) {
		t.Fatalf("implementation handoff history = %q, want approved plan", handoffContent)
	}
	if !strings.Contains(handoffContent, planModeFlowStructuredRelatedFilesFragment) {
		t.Fatalf("implementation handoff history = %q, want structured related files", handoffContent)
	}
	if strings.Contains(handoffContent, "Files: foo.go") {
		t.Fatalf("implementation handoff history = %q, should not use legacy Files label", handoffContent)
	}
}

func assertImplementationHistoryOmitsInvestigationContext(t *testing.T, history []api.Message) {
	t.Helper()
	for _, msg := range history {
		for _, fragment := range []string{
			"READ-ONLY ONLY",
			"Modification tools are FORBIDDEN",
			"You are in PLAN MODE (Investigation Phase).",
		} {
			if strings.Contains(msg.Content, fragment) {
				t.Fatalf("implementation history must not contain investigation context %q, got %q", fragment, msg.Content)
			}
		}
	}
}

func assertImplementationPreviousResponseIDCleared(t *testing.T, provider *planResponseContextProvider) {
	t.Helper()
	if len(provider.usedPreviousResponseIDs) != 2 {
		t.Fatalf("len(usedPreviousResponseIDs) = %d, want 2", len(provider.usedPreviousResponseIDs))
	}
	if provider.usedPreviousResponseIDs[1] != "" {
		t.Fatalf("implementation previous_response_id = %q, want empty", provider.usedPreviousResponseIDs[1])
	}
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

func TestChatCore_PlanModeApproval_StartsImplementationWithApprovedPlan(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := newPlanApprovalFlowProvider("")
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
	implementationHistory := planImplementationHistory(t, provider)
	if agent.SystemPrompt != baselineSystemPrompt {
		t.Fatalf("SystemPrompt should be restored after /plan approval")
	}
	if strings.Contains(agent.SystemPrompt, "You are in Plan Mode - producing a text plan") {
		t.Fatalf("planning prompt should not remain in SystemPrompt: %q", agent.SystemPrompt)
	}
	assertImplementationHistoryContainsApprovedPlanHandoff(t, implementationHistory)
	assertImplementationHistoryOmitsInvestigationContext(t, implementationHistory)
	if len(agent.History) != 2 {
		t.Fatalf("len(agent.History) = %d, want 2 for handoff user + implementation response", len(agent.History))
	}
	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	if len(agent.session.Messages) != 2 {
		t.Fatalf("len(agent.session.Messages) = %d, want 2 for handoff user + implementation response", len(agent.session.Messages))
	}
	if !strings.Contains(agent.session.Messages[0].Content, "Implement the approved plan now.") {
		t.Fatalf("session handoff = %q, want approved plan handoff", agent.session.Messages[0].Content)
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	loadedMessages := loaded.ToAPIMessages()
	if len(loadedMessages) != 2 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 2 after implementation handoff", len(loadedMessages))
	}
	if strings.Contains(out.String(), "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("plan approval should not create legacy marker, got %q", out.String())
	}
	if !strings.Contains(out.String(), planModeFlowImplementationStartText) {
		t.Fatalf("expected implementation start output, got %q", out.String())
	}
}

func TestChatCore_PlanModeApproval_DoesNotCarryPlanningResponseContextToImplementation(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := newPlanApprovalFlowProvider("")
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("first chatCore() error = %v", err)
	}
	assertImplementationPreviousResponseIDCleared(t, provider)
	assertImplementationHistoryOmitsInvestigationContext(t, planImplementationHistory(t, provider))
}

func TestChatCore_PlanModeApproval_ClearsPrePlanResponseContextForImplementation(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := newPlanApprovalFlowProvider("resp_before_plan")
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("first chatCore() error = %v", err)
	}
	assertPlanAndImplementationProviderCalls(t, provider)
	assertImplementationPreviousResponseIDCleared(t, provider)
}

func TestRetryChatRequest_PlanModeApprovalStartsImplementation(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := newPlanApprovalFlowProvider("")
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true
	req := &chatRequest{input: "implement feature", autoCompression: newAutoCompressionTurnState()}
	agent.prepareChatRequest(req)

	if err := agent.retryChatRequest(req); err != nil {
		t.Fatalf("retryChatRequest() error = %v", err)
	}
	implementationHistory := planImplementationHistory(t, provider)
	if agent.PlanModeEnabled {
		t.Fatal("PlanModeEnabled should be false after approval")
	}
	assertImplementationHistoryContainsApprovedPlanHandoff(t, implementationHistory)
	assertImplementationPreviousResponseIDCleared(t, provider)
}

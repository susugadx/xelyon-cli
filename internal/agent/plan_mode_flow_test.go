package agent

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
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
	systemPrompts           []string
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
	p.systemPrompts = append(p.systemPrompts, systemPrompt)
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

func assertImplementationSystemPromptOmitsPlanModePrompt(t *testing.T, provider *planResponseContextProvider) {
	t.Helper()
	assertPlanAndImplementationProviderCalls(t, provider)
	if len(provider.systemPrompts) != 2 {
		t.Fatalf("len(provider.systemPrompts) = %d, want 2", len(provider.systemPrompts))
	}
	if strings.Contains(provider.systemPrompts[1], "You are in Plan Mode - producing a text plan") {
		t.Fatalf("implementation system prompt should not contain planning prompt:\n%s", provider.systemPrompts[1])
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

func TestChatCore_PlanModeNoStepPlanCompletesWithoutRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{"```json\n" + `{
  "plan": {
    "findings": ["Existing behavior already satisfies the request"],
    "evidence": ["README.md documents it"],
    "constraints": ["Do not change CLI output"],
    "steps": []
  }
}` + "\n```"},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("investigate only", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1 without plan JSON retry", provider.callCount)
	}
	if !strings.Contains(out.String(), "No implementation steps needed.") {
		t.Fatalf("expected no-implementation output, got %q", out.String())
	}
	for _, want := range []string{
		"Investigation Result",
		"Existing behavior already satisfies the request",
		"README.md documents it",
		"Do not change CLI output",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected no-step plan details to include %q, got %q", want, out.String())
		}
	}
	assertPlanJSONRetryPromptNotAppended(t, agent)
	if status := agent.statusRef().getStatus(); status.State != StateWaitingInput {
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
	assertImplementationSystemPromptOmitsPlanModePrompt(t, provider)
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

func TestChatCore_PlanModeApproval_TaskSummaryShowsPlannedVerification(t *testing.T) {
	disableColors(t)

	targetPath := filepath.Join(t.TempDir(), "plan_followthrough.go")
	var out bytes.Buffer
	provider := &planResponseContextProvider{
		responses: []string{
			"```json\n" + `{
  "plan": {
    "summary": "Update implementation",
    "steps": [
      {
        "id": 1,
        "description": "Write implementation file",
        "tools": ["final_check_write"],
        "verification": [
          "go test ./internal/agent",
          "go test ./internal/agent",
          "make ci-check"
        ]
      }
    ]
  }
}` + "\n```",
			`{"tool":"final_check_write","args":{"path":"` + targetPath + `","content":"package main\n"}}`,
			"Done.",
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.registry().Register(&finalCheckWriteTool{})
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}

	output := out.String()
	for _, want := range []string{
		"Task Completed",
		"Planned verification",
		"go test ./internal/agent",
		"make ci-check",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	summaryStart := strings.LastIndex(output, "Task Completed")
	if summaryStart < 0 {
		t.Fatalf("output missing task summary:\n%s", output)
	}
	summaryOutput := output[summaryStart:]
	if strings.Count(summaryOutput, "go test ./internal/agent") != 1 {
		t.Fatalf("planned verification should be deduplicated in task summary, got output:\n%s", summaryOutput)
	}
}

func TestChatCore_PlanModeCancel_DisablesPlanModeForNextRequest(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &planResponseContextProvider{
		responses: []string{
			planHandoffTestApprovedPlanResponse(planHandoffTestApprovedStep),
			"normal response",
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "n\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("first chatCore() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount after cancel = %d, want 1", provider.callCount)
	}
	if !strings.Contains(out.String(), "Plan mode cancelled. No implementation started.") {
		t.Fatalf("expected cancel output, got %q", out.String())
	}
	if agent.PlanModeEnabled {
		t.Fatal("PlanModeEnabled should be false after cancel")
	}

	firstOutputLen := out.Len()
	if err := agent.chatCore("normal followup", nil, false); err != nil {
		t.Fatalf("second chatCore() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount after next request = %d, want 2", provider.callCount)
	}
	secondOutput := out.String()[firstOutputLen:]
	if strings.Contains(secondOutput, "Investigation phase - researching the codebase") {
		t.Fatalf("next request unexpectedly re-entered Plan Mode, output: %q", secondOutput)
	}
	if len(provider.histories) != 2 {
		t.Fatalf("len(provider.histories) = %d, want 2", len(provider.histories))
	}
	nextHistory := provider.histories[1]
	if len(nextHistory) == 0 {
		t.Fatal("normal followup history is empty")
	}
	nextUserContent := nextHistory[len(nextHistory)-1].Content
	if !strings.Contains(nextUserContent, "normal followup") {
		t.Fatalf("next request history = %q, want normal followup", nextUserContent)
	}
	if strings.Contains(nextUserContent, "PLAN MODE") || strings.Contains(nextUserContent, "Investigation Phase") {
		t.Fatalf("next request history = %q, should not contain plan investigation prompt", nextUserContent)
	}
}

func TestChatCore_PlanModeApproval_HandoffCarriesInvestigationSummaryFieldsOnly(t *testing.T) {
	disableColors(t)

	const rawInvestigationFragment = "RAW INVESTIGATION TRANSCRIPT SHOULD NOT CARRY"
	var out bytes.Buffer
	provider := &planResponseContextProvider{
		responses: []string{
			rawInvestigationFragment + `
Here is the approved plan:
` + "```json" + `
{
  "plan": {
    "summary": "Ship summarized plan handoff",
    "findings": ["plan_handoff.go owns implementation handoff"],
    "evidence": [
      "internal/agent/plan_handoff.go: normalModeInput",
      "internal/agent/plan_handoff_test.go: TestPlanModeImplementationHandoff_NormalModeInputIncludesApprovedPlan"
    ],
    "constraints": ["Do not carry raw investigation history"],
    "steps": [
      {
        "id": 1,
        "description": "Update handoff summary fields",
        "tools": ["str_replace"],
        "files": ["internal/agent/plan_handoff.go"]
      }
    ]
  }
}
` + "```",
			planModeFlowImplementationResponse,
		},
	}
	agent := newPlanRequestTestAgent(t, provider, "1\n", &out)
	agent.PlanModeEnabled = true

	if err := agent.chatCore("implement feature", nil, false); err != nil {
		t.Fatalf("chatCore() error = %v", err)
	}

	implementationHistory := planImplementationHistory(t, provider)
	handoffContent := implementationHistory[len(implementationHistory)-1].Content
	for _, want := range []string{
		"Findings:",
		" - plan_handoff.go owns implementation handoff",
		"Evidence:",
		" - internal/agent/plan_handoff.go: normalModeInput",
		" - internal/agent/plan_handoff_test.go: TestPlanModeImplementationHandoff_NormalModeInputIncludesApprovedPlan",
		"Constraints:",
		" - Do not carry raw investigation history",
	} {
		if !strings.Contains(handoffContent, want) {
			t.Fatalf("implementation handoff = %q, want fragment %q", handoffContent, want)
		}
	}
	if strings.Contains(handoffContent, rawInvestigationFragment) {
		t.Fatalf("implementation handoff should not carry raw investigation text, got %q", handoffContent)
	}
	assertImplementationHistoryOmitsInvestigationContext(t, implementationHistory)
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

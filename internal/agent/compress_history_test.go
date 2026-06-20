package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type capturingMockProvider struct {
	capturedHistory      []api.Message
	capturedSystemPrompt string
	capturedContext      context.Context
}

func (m *capturingMockProvider) Name() string                   { return "test" }
func (m *capturingMockProvider) SupportsImages() bool           { return false }
func (m *capturingMockProvider) IsFunctionCallingEnabled() bool { return true }
func (m *capturingMockProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, _ string) (string, error) {
	m.capturedContext = ctx
	m.capturedSystemPrompt = systemPrompt
	m.capturedHistory = history
	return compressionSummaryResponseForHistory(history, "Summary of conversation"), nil
}
func (m *capturingMockProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func TestCompressHistory_PrePrunesBeforeSummary(t *testing.T) {
	provider := &capturingMockProvider{}
	agent := NewAgent("test-model", provider, false)

	large := makeLargeContent(60)
	agent.History = []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
		{Role: "user", Content: "turn 4"},
		{Role: "assistant", Content: "a4"},
		{Role: "user", Content: "turn 5"},
		{Role: "assistant", Content: "a5"},
		{Role: "user", Content: "turn 6"},
		{Role: "assistant", Content: "a6"},
		{Role: "user", Content: "turn 7"},
		{Role: "assistant", Content: "a7"},
	}

	err := agent.CompressHistory(4)
	if err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	if len(provider.capturedHistory) == 0 {
		t.Fatal("ChatWithTools was not called")
	}
	capturedPrompt := provider.capturedHistory[0].Content
	if !strings.Contains(capturedPrompt, "truncated") {
		t.Error("CompressHistory() should pre-prune old tool results before BuildSummaryPrompt")
	}
}

func TestCompressHistory_PassesContinuationV1SystemPrompt(t *testing.T) {
	provider := &capturingMockProvider{}
	agent := NewAgent("test-model", provider, false)
	t.Cleanup(agent.Cleanup)
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "latest"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if !strings.Contains(provider.capturedSystemPrompt, "xelyon.continuation.v1") {
		t.Fatalf("summary system prompt = %q, want continuation v1 contract", provider.capturedSystemPrompt)
	}
	if len(provider.capturedHistory) != 1 || provider.capturedHistory[0].Role != "user" {
		t.Fatalf("summary history = %#v, want one user transcript prompt", provider.capturedHistory)
	}
}

func TestCompressHistory_InvalidNestedContinuationKeepsOriginalHistory(t *testing.T) {
	provider := &compressionTestProvider{
		name:    "openai",
		summary: `{"schema_version":"xelyon.continuation.v1","goal":"fix compression","acceptance_criteria":[],"explicit_constraints":[],"material_assumptions":[],"decisions":[{"decision":"keep strict parser","evidence":[]}],"files_changed":[],"verification":[],"open_work":[],"blockers":[],"do_not_repeat":[],"relevant_instruction_refs":[]}`,
	}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	originalHistory := []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "older"},
		{Role: "user", Content: "latest"},
	}
	agent.History = append([]api.Message(nil), originalHistory...)

	err := agent.CompressHistory(1)
	if err == nil || !strings.Contains(err.Error(), "decisions[0] missing keys: reason") {
		t.Fatalf("CompressHistory() error = %v, want nested continuation validation error", err)
	}
	if len(agent.History) != len(originalHistory) {
		t.Fatalf("len(agent.History) = %d, want original %d", len(agent.History), len(originalHistory))
	}
	for i := range originalHistory {
		if agent.History[i].Role != originalHistory[i].Role || agent.History[i].Content != originalHistory[i].Content {
			t.Fatalf("agent.History[%d] = %#v, want %#v", i, agent.History[i], originalHistory[i])
		}
	}
}

func TestCompressHistory_UsesCompressionModelDefault(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: "message 1"},
		{Role: "assistant", Content: "message 2"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if provider.capturedChatModel != "gpt-5.4-mini" {
		t.Fatalf("CompressHistory() model = %q, want %q", provider.capturedChatModel, "gpt-5.4-mini")
	}
}

func TestCompressHistory_SuppressesAssistantUpdatesForSummaryRequest(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	cfg := config.DefaultConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesVerbose

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "latest"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if provider.capturedChatUpdateMode != api.AssistantUpdatesOff {
		t.Fatalf("summary request assistant update mode = %q, want %q", provider.capturedChatUpdateMode, api.AssistantUpdatesOff)
	}
}

func TestCompressHistory_DoesNotSendCurrentTaskStateActiveContext(t *testing.T) {
	provider := &capturingMockProvider{}
	agent := NewAgent("gpt-5.4", provider, false)
	t.Cleanup(agent.Cleanup)
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)
	agent.History = []api.Message{
		{Role: "user", Content: "old message"},
		{Role: "assistant", Content: "old response"},
		{Role: "user", Content: "latest message"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if got := api.ActiveContextBlocksFromContext(provider.capturedContext); got != nil {
		t.Fatalf("compression summary active context = %#v, want nil", got)
	}
	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatal("CompressHistory() should not reset the runtime task ledger")
	}
}

func TestCompressHistory_ClearsProviderHistoryReductionTaskLedgerOnSuccess(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	fixture := newProviderHistoryStaleLedgerFixture()
	seedProviderHistoryReductionStaleLedgerEvidence(t, agent, fixture)
	assertTaskLedgerPreserved(t, agent, "test setup")
	agent.History = []api.Message{
		{Role: "user", Content: "old message"},
		{Role: "assistant", Content: "old response"},
		{Role: "user", Content: "latest message"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	assertTaskLedgerReset(t, agent, "CompressHistory provider history reduction success")
	agent.History = fixture.History
	assertProviderHistoryReductionDoesNotUseStaleLedgerEvidence(t, agent, fixture)
}

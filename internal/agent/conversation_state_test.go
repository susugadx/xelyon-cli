package agent

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type conversationStateTestProvider struct {
	responseID string
}

func (p *conversationStateTestProvider) Name() string { return "test" }

func (p *conversationStateTestProvider) SupportsImages() bool { return false }

func (p *conversationStateTestProvider) IsFunctionCallingEnabled() bool { return false }

func (p *conversationStateTestProvider) ChatWithTools(_ context.Context, _ string, _ []api.Message, _ string) (string, error) {
	return "done", nil
}

func (p *conversationStateTestProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "done", nil
}

func (p *conversationStateTestProvider) HasCachedResponseID() bool {
	return p.responseID != ""
}

func (p *conversationStateTestProvider) SetResponseID(id string) {
	p.responseID = id
}

func (p *conversationStateTestProvider) GetResponseID() string {
	return p.responseID
}

func TestAgent_ResetConversationState_ClearsRuntimeAndSessionState(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)

	agent.History = []api.Message{{Role: "assistant", Content: "old response"}}
	agent.lastOutputs = []string{"old response"}
	agent.compactedItems = []api.InputItem{{Type: "compacted", Data: "compressed"}}
	agent.isCompactedMode = true
	provider.SetResponseID("resp_old")

	if agent.session == nil {
		t.Fatal("agent.session = nil")
	}
	agent.session.AddMessage("user", "old request", agent.CurrentModel)
	agent.session.CompactedItems = []history.CompactedItem{{Type: "compacted", Data: "compressed"}}
	agent.session.IsCompactedMode = true
	agent.session.ResponseID = "resp_old"
	agent.persistSession()

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}

	if len(agent.History) != 0 {
		t.Fatalf("len(agent.History) = %d, want 0", len(agent.History))
	}
	if len(agent.lastOutputs) != 0 {
		t.Fatalf("len(agent.lastOutputs) = %d, want 0", len(agent.lastOutputs))
	}
	if agent.IsCompactedMode() {
		t.Fatal("IsCompactedMode() = true, want false")
	}
	if len(agent.GetCompactedItems()) != 0 {
		t.Fatalf("len(GetCompactedItems()) = %d, want 0", len(agent.GetCompactedItems()))
	}
	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want empty", provider.GetResponseID())
	}

	loaded, err := agent.storage.Load(agent.session.ID)
	if err != nil {
		t.Fatalf("storage.Load() error = %v", err)
	}
	if len(loaded.ToAPIMessages()) != 0 {
		t.Fatalf("len(loaded.ToAPIMessages()) = %d, want 0", len(loaded.ToAPIMessages()))
	}
	if loaded.IsCompactedMode {
		t.Fatal("loaded.IsCompactedMode = true, want false")
	}
	if len(loaded.CompactedItems) != 0 {
		t.Fatalf("len(loaded.CompactedItems) = %d, want 0", len(loaded.CompactedItems))
	}
	if loaded.ResponseID != "" {
		t.Fatalf("loaded.ResponseID = %q, want empty", loaded.ResponseID)
	}
}

func TestAgent_ResetConversationState_ClearsModelFacingTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "openai"}, currentTaskStateOpenAIResponses, &out)
	assertCurrentTaskLedgerPreserved(t, agent, "test setup")

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}
	assertCurrentTaskLedgerReset(t, agent, "resetConversationState with provider-facing current task state")
}

func TestAgent_ResetConversationState_PreservesTaskLedgerWhenCurrentTaskStateProviderDoesNotConsumeIt(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "deepseek"}, currentTaskStateDeepSeek, &out)

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}
	assertCurrentTaskLedgerPreserved(t, agent, "resetConversationState with non-consuming provider")
}

func TestAgent_ResetConversationState_PreservesTaskLedgerWhenCurrentTaskStateContextDisabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}
	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatal("task ledger was reset even though current task state context is disabled")
	}
}

func TestAgent_ApplyLoadedSession_ClearsModelFacingTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "openai"}, currentTaskStateOpenAIResponses, &out)

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded request", "test-model")

	agent.applyLoadedSession(session)

	assertCurrentTaskLedgerReset(t, agent, "applyLoadedSession with provider-facing current task state")
	if got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background())); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil after session load resets task ledger", got)
	}
}

func TestAgent_ApplyLoadedSession_PreservesTaskLedgerWhenCurrentTaskStateProviderDoesNotConsumeIt(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "deepseek"}, currentTaskStateDeepSeek, &out)

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded request", "test-model")

	agent.applyLoadedSession(session)

	assertCurrentTaskLedgerPreserved(t, agent, "applyLoadedSession with non-consuming provider")
}

func TestAgent_ApplyLoadedSession_PreservesTaskLedgerWhenCurrentTaskStateContextDisabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded request", "test-model")

	agent.applyLoadedSession(session)

	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatal("task ledger was reset on session load even though current task state context is disabled")
	}
}

func TestAgent_RestoreSessionConversation_ReplacesRuntimeMirrors(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)

	agent.History = []api.Message{{Role: "assistant", Content: "stale response"}}
	agent.lastOutputs = []string{"stale output"}
	agent.compactedItems = []api.InputItem{{Type: "compacted", Data: "stale"}}
	agent.isCompactedMode = true
	provider.SetResponseID("resp_stale")

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded request", "test-model")

	agent.restoreSessionConversation(session)

	if agent.session != session {
		t.Fatal("agent.session should be replaced with loaded session")
	}
	if len(agent.History) != 1 || agent.History[0].Content != "loaded request" {
		t.Fatalf("agent.History = %#v, want loaded session messages", agent.History)
	}
	if agent.IsCompactedMode() {
		t.Fatal("IsCompactedMode() = true, want false after loading non-compacted session")
	}
	if len(agent.GetCompactedItems()) != 0 {
		t.Fatalf("len(GetCompactedItems()) = %d, want 0", len(agent.GetCompactedItems()))
	}
	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want empty after loading session without response ID", provider.GetResponseID())
	}
	if len(agent.lastOutputs) != 0 {
		t.Fatalf("len(agent.lastOutputs) = %d, want 0 after session restore", len(agent.lastOutputs))
	}
}

func TestAgent_RestoreSessionConversation_RestoresCompactionAndResponseID(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)

	session := history.NewSession("test-model")
	session.CompactedItems = []history.CompactedItem{{Type: "compacted", Data: "compressed"}}
	session.IsCompactedMode = true
	session.ResponseID = "resp_loaded"

	agent.restoreSessionConversation(session)

	if !agent.IsCompactedMode() {
		t.Fatal("IsCompactedMode() = false, want true")
	}
	if len(agent.GetCompactedItems()) != 1 {
		t.Fatalf("len(GetCompactedItems()) = %d, want 1", len(agent.GetCompactedItems()))
	}
	if got := agent.GetCompactedItems()[0].Data; !strings.Contains(got, "compressed") {
		t.Fatalf("compacted item data = %q, want restored data", got)
	}
	if provider.GetResponseID() != "resp_loaded" {
		t.Fatalf("provider response ID = %q, want resp_loaded", provider.GetResponseID())
	}
}

func TestAgent_RestoreSessionConversation_PreservesRichCompactedInputItems(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)

	items := []history.CompactedItem{
		{Type: "function_call", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`},
		{Type: "function_call_output", CallID: "call_1", Output: "README contents"},
	}
	session := history.NewSession("test-model")
	session.SetCompactedState(items, true)

	agent.restoreSessionConversation(session)

	if !reflect.DeepEqual(agent.GetCompactedItems(), []api.InputItem(items)) {
		t.Fatalf("GetCompactedItems() = %#v, want %#v", agent.GetCompactedItems(), items)
	}
}

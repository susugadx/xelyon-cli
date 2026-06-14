package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

type conversationStateTestProvider struct {
	name       string
	responseID string
}

func (p *conversationStateTestProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "test"
}

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

func TestAgent_StartNewSession_SaveFailurePreservesActiveRuntimeState(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{name: "openai"}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldSession := agent.session
	oldSessionID := oldSession.ID
	oldStats := agent.Stats
	agent.History = []api.Message{{Role: "user", Content: "active request"}}
	provider.SetResponseID("resp_active")

	beforeStartNewSessionMetadataSaveForTest = func() {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("UserHomeDir() error = %v", err)
		}
		metadataDir := filepath.Join(home, ".xelyon", "history", "metadata")
		if err := os.RemoveAll(metadataDir); err != nil {
			t.Fatalf("RemoveAll(metadataDir) error = %v", err)
		}
		if err := os.WriteFile(metadataDir, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile(metadataDir) error = %v", err)
		}
	}
	t.Cleanup(func() { beforeStartNewSessionMetadataSaveForTest = nil })

	_, err := agent.StartNewSession()
	if err == nil {
		t.Fatal("StartNewSession() error = nil, want metadata save failure")
	}
	if !strings.Contains(err.Error(), "save new session metadata") {
		t.Fatalf("StartNewSession() error = %v, want save new session metadata", err)
	}
	if agent.session != oldSession || agent.session.ID != oldSessionID {
		t.Fatalf("agent.session = %#v, want old session %s", agent.session, oldSessionID)
	}
	if len(agent.History) != 1 || agent.History[0].Content != "active request" {
		t.Fatalf("agent.History = %#v, want active request preserved", agent.History)
	}
	if got := provider.GetResponseID(); got != "resp_active" {
		t.Fatalf("provider response ID = %q, want resp_active", got)
	}
	if agent.Stats != oldStats {
		t.Fatal("agent.Stats pointer changed after failed StartNewSession")
	}
}

func TestAgent_ResetConversationState_ClearsModelFacingTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "openai"}, activeContextOpenAIResponses, &out)
	assertCurrentTaskLedgerPreserved(t, agent, "test setup")

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}
	assertCurrentTaskLedgerReset(t, agent, "resetConversationState with provider-facing current task state")
}

func TestAgent_ResetConversationState_ClearsProviderHistoryReductionTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	fixture := newProviderHistoryStaleLedgerFixture()
	seedProviderHistoryReductionStaleLedgerEvidence(t, agent, fixture)
	assertTaskLedgerPreserved(t, agent, "test setup")

	if err := agent.resetConversationState(); err != nil {
		t.Fatalf("resetConversationState() error = %v", err)
	}

	assertTaskLedgerReset(t, agent, "resetConversationState with provider history reduction")
	agent.History = fixture.History
	assertProviderHistoryReductionDoesNotUseStaleLedgerEvidence(t, agent, fixture)
}

func TestAgent_ResetConversationState_PreservesTaskLedgerWhenCurrentTaskStateProviderDoesNotConsumeIt(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "unsupported"}, activeContextUnsupported, &out)

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

func TestAgent_ResumeSession_ModelSwitchPreservesOutgoingSessionResponseContext(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{name: "openai"}
	agent := newChatRequestTestAgent(t, provider, &out)
	outgoingID := agent.session.ID
	agent.session.AddMessage("user", "current request", agent.CurrentModel)
	provider.SetResponseID("resp_current")

	target := history.NewSession("gpt-5.5")
	target.ProviderName = "openai"
	target.ProviderConfigKey = "openai"
	target.AddMessage("user", "saved request", "gpt-5.5")
	if err := agent.storage.Save(target); err != nil {
		t.Fatalf("Save(target) error = %v", err)
	}

	if _, err := agent.ResumeSession(target.ID); err != nil {
		t.Fatalf("ResumeSession() error = %v", err)
	}

	outgoing, err := agent.storage.Load(outgoingID)
	if err != nil {
		t.Fatalf("Load(outgoing) error = %v", err)
	}
	if outgoing.Model != "gpt-5.4" || outgoing.ProviderName != "openai" || outgoing.ProviderConfigKey != "openai" {
		t.Fatalf("outgoing identity = (%q, %q, %q), want openai/gpt-5.4", outgoing.ProviderName, outgoing.ProviderConfigKey, outgoing.Model)
	}
	if outgoing.ResponseID != "resp_current" ||
		outgoing.ResponseModel != "gpt-5.4" ||
		outgoing.ResponseProviderName != "openai" ||
		outgoing.ResponseProviderConfigKey != "openai" {
		t.Fatalf("outgoing response context = (%q, %q, %q, %q), want resp_current/openai/gpt-5.4",
			outgoing.ResponseID,
			outgoing.ResponseModel,
			outgoing.ResponseProviderName,
			outgoing.ResponseProviderConfigKey,
		)
	}
	if agent.session == nil || agent.session.ID != target.ID {
		t.Fatalf("agent.session.ID = %q, want target %s", agent.session.ID, target.ID)
	}
}

func TestAgent_ApplyLoadedSession_ClearsModelFacingTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "openai"}, activeContextOpenAIResponses, &out)

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded request", "test-model")

	agent.applyLoadedSession(session)

	assertCurrentTaskLedgerReset(t, agent, "applyLoadedSession with provider-facing current task state")
	if got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background())); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil after session load resets task ledger", got)
	}
}

func TestAgent_ApplyLoadedSession_ClearsProviderHistoryReductionTaskLedger(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &conversationStateTestProvider{}
	agent := newChatRequestTestAgent(t, provider, &out)
	fixture := newProviderHistoryStaleLedgerFixture()
	seedProviderHistoryReductionStaleLedgerEvidence(t, agent, fixture)
	assertTaskLedgerPreserved(t, agent, "test setup")

	session := history.NewSession("test-model")
	for _, msg := range fixture.History {
		session.AddMessageFromAPI(msg, "test-model")
	}

	agent.applyLoadedSession(session)

	assertTaskLedgerReset(t, agent, "applyLoadedSession with provider history reduction")
	assertProviderHistoryReductionDoesNotUseStaleLedgerEvidence(t, agent, fixture)
}

func TestAgent_ApplyLoadedSession_PreservesTaskLedgerWhenCurrentTaskStateProviderDoesNotConsumeIt(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newCurrentTaskStateAgent(t, &mockProvider{name: "unsupported"}, activeContextUnsupported, &out)

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

func TestAgent_ResumeStartupSession_FailureDoesNotPersistBootstrapSessionOnCleanup(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &conversationStateTestProvider{}, &out)
	bootstrapID := agent.session.ID

	if _, err := agent.ResumeStartupSession("missing-session"); err == nil {
		t.Fatal("ResumeStartupSession() error = nil, want missing session error")
	}
	agent.Cleanup()

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(ListSessions()) = %d, want 0; bootstrap session %s should not be persisted", len(sessions), bootstrapID)
	}
}

func TestAgent_ResumeStartupSession_SuccessPersistsOnlyLoadedSessionOnCleanup(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &conversationStateTestProvider{}, &out)
	bootstrapID := agent.session.ID

	loadedSession := history.NewSession("gpt-5.4")
	loadedSession.AddMessage("user", "saved request", "gpt-5.4")
	if err := agent.storage.Save(loadedSession); err != nil {
		t.Fatalf("Save(loadedSession) error = %v", err)
	}

	resumed, err := agent.ResumeStartupSession(loadedSession.ID)
	if err != nil {
		t.Fatalf("ResumeStartupSession() error = %v", err)
	}
	if resumed.ID != loadedSession.ID {
		t.Fatalf("resumed.ID = %q, want %q", resumed.ID, loadedSession.ID)
	}
	if agent.session.ID != loadedSession.ID {
		t.Fatalf("agent.session.ID = %q, want loaded session %q", agent.session.ID, loadedSession.ID)
	}
	agent.Cleanup()

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(ListSessions()) = %d, want only loaded session; bootstrap session %s should not be persisted", len(sessions), bootstrapID)
	}
	if sessions[0].ID != loadedSession.ID {
		t.Fatalf("sessions[0].ID = %q, want loaded session %q", sessions[0].ID, loadedSession.ID)
	}
}

func TestAgent_ResumeStartupLastSession_SuccessDoesNotPersistBootstrapSession(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &conversationStateTestProvider{}, &out)
	bootstrapID := agent.session.ID

	loadedSession := history.NewSession("gpt-5.4")
	loadedSession.AddMessage("user", "saved request", "gpt-5.4")
	if err := agent.storage.Save(loadedSession); err != nil {
		t.Fatalf("Save(loadedSession) error = %v", err)
	}

	resumed, err := agent.ResumeStartupLastSession(history.ResumeListOptions{})
	if err != nil {
		t.Fatalf("ResumeStartupLastSession() error = %v", err)
	}
	if resumed.ID != loadedSession.ID {
		t.Fatalf("resumed.ID = %q, want %q", resumed.ID, loadedSession.ID)
	}
	agent.Cleanup()

	sessions, err := agent.storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != loadedSession.ID {
		t.Fatalf("sessions = %#v, want only loaded session %s; bootstrap %s must not persist", sessions, loadedSession.ID, bootstrapID)
	}
}

func TestAgent_ResumeStartupLastSession_LoadFailureIsNotNoSessions(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &conversationStateTestProvider{}, &out)

	missingBodySession := history.NewSession("gpt-5.4")
	missingBodySession.AddMessage("user", "saved request", "gpt-5.4")
	if err := agent.storage.Save(missingBodySession); err != nil {
		t.Fatalf("Save(missingBodySession) error = %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	if err := os.Remove(filepath.Join(home, ".xelyon", "history", missingBodySession.ID+".jsonl")); err != nil {
		t.Fatalf("Remove(session body) error = %v", err)
	}

	_, err = agent.ResumeStartupLastSession(history.ResumeListOptions{})
	if err == nil {
		t.Fatal("ResumeStartupLastSession() error = nil, want load failure")
	}
	if errors.Is(err, history.ErrNoResumeSessions) {
		t.Fatalf("ResumeStartupLastSession() error = %v, must not be ErrNoResumeSessions when metadata exists but load fails", err)
	}
	if !strings.Contains(err.Error(), "load session") {
		t.Fatalf("ResumeStartupLastSession() error = %v, want load session", err)
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

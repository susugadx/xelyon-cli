package agent

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRequestContext_CurrentTaskStateContextDefaultOff(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil when option is off", got)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsInheritedBlocksWhenDefaultOff(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want inherited blocks cleared when option is off", got)
	}
}

func TestRequestContext_CurrentTaskStateContextNilLedgerNoop(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{EnableCurrentTaskStateContext: true},
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for nil TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextEmptyLedgerNoop(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: ledger.NewStoreWithRoot(t.TempDir()),
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for empty TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsInheritedBlocksWhenLedgerIsEmpty(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: ledger.NewStoreWithRoot(t.TempDir()),
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want inherited blocks cleared when ledger is empty", got)
	}
}

func TestRequestContext_CurrentTaskStateContextIncludesSnapshotWithoutMutatingHistory(t *testing.T) {
	session := history.NewSession("gpt-5.4")
	session.AddMessage("user", "persisted", "gpt-5.4")
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "live"}},
		agentConversationState: agentConversationState{
			session: session,
		},
	}
	beforeHistory := append([]api.Message(nil), agent.History...)
	beforeSessionMessages := append([]history.MessageEntry(nil), session.Messages...)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if len(got) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(got))
	}
	if got[0].Name != currentTaskStateActiveContextName {
		t.Fatalf("Name = %q, want %q", got[0].Name, currentTaskStateActiveContextName)
	}
	for _, want := range []string{
		ledger.CurrentTaskStateStartMarker,
		"Last passed tests:",
		"go test ./internal/ledger",
		ledger.CurrentTaskStateEndMarker,
	} {
		if !strings.Contains(got[0].Content, want) {
			t.Fatalf("active context content missing %q:\n%s", want, got[0].Content)
		}
	}

	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("History mutated:\ngot:  %#v\nwant: %#v", agent.History, beforeHistory)
	}
	if !reflect.DeepEqual(session.Messages, beforeSessionMessages) {
		t.Fatalf("session.Messages mutated:\ngot:  %#v\nwant: %#v", session.Messages, beforeSessionMessages)
	}
}

func TestRequestContextWithoutActiveContextClearsOwnedAndInheritedBlocks(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContextWithoutActiveContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for internal model request context", got)
	}
}

func TestEstimateTokens_IncludesModelFacingCurrentTaskStateContext(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	blocks := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if len(blocks) != 1 {
		t.Fatalf("len(ActiveContextBlocksFromContext()) = %d, want 1", len(blocks))
	}

	wantActive := token.EstimateTokenCountForModel(agent.CurrentModel, blocks[0].Content)
	if got := agent.EstimateActiveContextTokens(); got != wantActive {
		t.Fatalf("EstimateActiveContextTokens() = %d, want %d", got, wantActive)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello") +
		wantActive
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history + active context = %d", got, wantTotal)
	}
}

func TestEstimateTokens_DoesNotCountCurrentTaskStateForProvidersThatDoNotConsumeIt(t *testing.T) {
	agent := &Agent{
		CurrentModel:      "deepseek-chat",
		ProviderName:      "deepseek",
		ProviderConfigKey: "deepseek",
		SystemPrompt:      "system",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "hello"}},
	}

	if got := agent.EstimateActiveContextTokens(); got != 0 {
		t.Fatalf("EstimateActiveContextTokens() = %d, want 0 for non-Responses provider", got)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) +
		token.EstimateTokenCountForModel(agent.CurrentModel, "hello")
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + history = %d", got, wantTotal)
	}
}

func TestHandleTokensCommand_ShowsCurrentTaskStateContextTokens(t *testing.T) {
	var out bytes.Buffer
	agent := &Agent{
		CurrentModel:      "gpt-5.4",
		ProviderName:      "openai",
		ProviderConfigKey: "openai",
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
			UI:         ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleTokensCommand(agent) {
		t.Fatal("handleTokensCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "Active Context:") {
		t.Fatalf("tokens output missing active context row:\n%s", out.String())
	}
}

func TestPrepareChatRequest_CurrentTaskStateContextClearsResponseContextBeforeSessionSave(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &mockResponseIDProvider{mockProvider: mockProvider{name: "openai"}, responseID: "resp_old"}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.Runtime.Options.EnableCurrentTaskStateContext = true
	agent.Runtime.TaskLedger = newTaskLedgerWithPassedTest(t)
	agent.session.ApplyResponseContext("resp_old", agent.CurrentModel, agent.ProviderName, agent.ProviderConfigKey)

	agent.prepareChatRequest(&chatRequest{input: "hello"})

	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want cleared before active-context request", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared before session save", agent.session.ResponseID)
	}
	if len(agent.session.Messages) == 0 || agent.session.Messages[len(agent.session.Messages)-1].Content != "hello" {
		t.Fatalf("session messages = %#v, want appended user message", agent.session.Messages)
	}
}

func contextWithInheritedActiveContext() context.Context {
	return api.WithActiveContextBlocks(context.Background(), []api.ActiveContextBlock{{
		Name:    currentTaskStateActiveContextName,
		Content: "parent snapshot",
	}})
}

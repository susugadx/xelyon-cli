package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func TestRequestContext_CurrentTaskStateContextDefaultOff(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

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
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

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
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for nil TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextEmptyLedgerNoop(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: taskstate.NewStoreWithRoot(t.TempDir()),
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

	got := api.ActiveContextBlocksFromContext(agent.requestContext(context.Background()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for empty TaskLedger", got)
	}
}

func TestRequestContext_CurrentTaskStateContextClearsInheritedBlocksWhenLedgerIsEmpty(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: taskstate.NewStoreWithRoot(t.TempDir()),
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

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
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)
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
		taskstate.CurrentTaskStateStartMarker,
		"Last passed tests:",
		"go test ./internal/taskstate",
		taskstate.CurrentTaskStateEndMarker,
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

func TestRequestContext_CurrentTaskStateContextClearsBlocksWhenProviderDoesNotConsumeIt(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyActiveContextProviderFixture(agent, activeContextUnsupported)

	if len(agent.buildActiveContextBlocks()) == 0 {
		t.Fatal("test setup produced empty active context")
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for provider that does not consume active context", got)
	}
}

func TestRequestContextWithoutActiveContextClearsOwnedAndInheritedBlocks(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)
	if len(agent.providerFacingActiveContextBlocks()) == 0 {
		t.Fatal("test setup produced empty provider-facing active context")
	}

	got := api.ActiveContextBlocksFromContext(agent.requestContextWithoutActiveContext(contextWithInheritedActiveContext()))
	if got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want nil for internal model request context", got)
	}
}

func contextWithInheritedActiveContext() context.Context {
	return api.WithActiveContextBlocks(context.Background(), []api.ActiveContextBlock{{
		Name:    currentTaskStateActiveContextName,
		Content: "parent snapshot",
	}})
}

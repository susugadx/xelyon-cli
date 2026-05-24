package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestModelInputAssemblyPlan_DefaultOffKeepsCompactedInputAndOmitsActiveContext(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		agentConversationState: agentConversationState{
			compactedItems:  []api.InputItem{{Type: "compacted", Data: "compact-data"}},
			isCompactedMode: true,
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)

	plan := agent.modelInputAssemblyPlan()
	if len(plan.CompactedInput) != 1 || plan.CompactedInput[0].Data != "compact-data" {
		t.Fatalf("CompactedInput = %#v, want current compacted input", plan.CompactedInput)
	}
	if len(plan.ActiveContextBlocks) != 0 {
		t.Fatalf("ActiveContextBlocks = %#v, want none when current task state context is off", plan.ActiveContextBlocks)
	}

	plan.CompactedInput[0].Data = "mutated"
	if agent.compactedItems[0].Data != "compact-data" {
		t.Fatalf("modelInputAssemblyPlan exposed compactedItems backing storage: %#v", agent.compactedItems)
	}

	ctx := agent.requestContext(contextWithInheritedActiveContext())
	if got := api.CompactedInputItemsFromContext(ctx); len(got) != 1 || got[0].Data != "compact-data" {
		t.Fatalf("CompactedInputItemsFromContext() = %#v, want compact-data", got)
	}
	if got := api.ActiveContextBlocksFromContext(ctx); got != nil {
		t.Fatalf("ActiveContextBlocksFromContext() = %#v, want inherited active context cleared", got)
	}
}

func TestModelInputAssemblyPlan_ActiveContextOnDoesNotMutateRawConversation(t *testing.T) {
	session := history.NewSession("gpt-5.4")
	session.AddMessage("user", "persisted user message", "gpt-5.4")
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options:    RuntimeOptions{EnableCurrentTaskStateContext: true},
			TaskLedger: newTaskLedgerWithPassedTest(t),
		},
		History: []api.Message{{Role: "user", Content: "live user message"}},
		agentConversationState: agentConversationState{
			session: session,
		},
	}
	applyActiveContextProviderFixture(agent, activeContextOpenAIResponses)
	beforeHistoryLen := len(agent.History)
	beforeSessionLen := len(session.Messages)

	plan := agent.modelInputAssemblyPlan()
	if len(plan.ActiveContextBlocks) != 1 {
		t.Fatalf("len(ActiveContextBlocks) = %d, want 1", len(plan.ActiveContextBlocks))
	}
	if len(agent.History) != beforeHistoryLen {
		t.Fatalf("History length = %d, want %d", len(agent.History), beforeHistoryLen)
	}
	if len(session.Messages) != beforeSessionLen {
		t.Fatalf("session messages = %d, want %d", len(session.Messages), beforeSessionLen)
	}
}

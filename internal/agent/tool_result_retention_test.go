package agent

import (
	"bytes"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestToolResultRetentionPolicy_DefaultKeepsAllRawToolResultStores(t *testing.T) {
	decision := defaultToolResultRetentionPolicy().Decide(toolResultRetentionInput{
		ToolCall: &tools.ToolCall{ID: "call_1", Tool: "read_file"},
	})

	if !decision.KeepHistory {
		t.Fatal("KeepHistory = false, want true")
	}
	if !decision.KeepSessionConversation {
		t.Fatal("KeepSessionConversation = false, want true")
	}
	if !decision.KeepSessionToolExecution {
		t.Fatal("KeepSessionToolExecution = false, want true")
	}
}

func TestNormalToolResultRetentionKeepsHistorySessionAndToolExecution(t *testing.T) {
	agent := newToolResultRetentionTestAgent(t)
	handler := newNormalModeToolResultHandler(&TurnRunner{agent: agent}, &normalModeState{turnMutations: turnsupport.NewMutationState()})
	toolCall := retentionTestToolCall("call_1", "README.md")

	handler.Handle(toolCall, toolruntime.Result{Result: "README contents"})

	assertRetainedToolResult(t, agent, toolResultAssertion{
		historyIndex:              0,
		sessionToolExecutionIndex: 0,
		sessionConversationIndex:  1,
		toolCallID:                "call_1",
		content:                   "README contents",
	})
}

func TestParallelAndBatchToolResultRetentionKeepsPerToolResultsThroughCallback(t *testing.T) {
	agent := newToolResultRetentionTestAgent(t)
	handler := newNormalModeToolResultHandler(&TurnRunner{agent: agent}, &normalModeState{turnMutations: turnsupport.NewMutationState()})
	toolCalls := []*tools.ToolCall{
		retentionTestToolCall("call_1", "README.md"),
		retentionTestToolCall("call_2", "main.go"),
	}
	state := toolruntime.NewParallelCallState(toolCalls)
	state.Entries[0] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusExecute}
	state.Results[0] = toolruntime.Result{Result: "README contents"}
	state.Entries[1] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusBatched}
	state.Results[1] = toolruntime.Result{Result: "main contents"}

	agent.deliverToolExecutionResults(state, func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		handler.Handle(tc, result)
	})

	assertRetainedToolResult(t, agent, toolResultAssertion{
		historyIndex:              0,
		sessionToolExecutionIndex: 0,
		sessionConversationIndex:  1,
		toolCallID:                "call_1",
		content:                   "README contents",
	})
	assertRetainedToolResult(t, agent, toolResultAssertion{
		historyIndex:              1,
		sessionToolExecutionIndex: 2,
		sessionConversationIndex:  3,
		toolCallID:                "call_2",
		content:                   "main contents",
	})
}

func TestHeadlessToolResultRetentionKeepsProviderHistoryWithoutSessionPersistence(t *testing.T) {
	agent := newToolResultRetentionTestAgent(t)
	toolCall := retentionTestToolCall("call_1", "README.md")

	appendHeadlessToolResultToHistory(agent, toolCall, "README contents")

	assertToolResultHistory(t, agent, toolResultAssertion{
		historyIndex: 0,
		toolCallID:   "call_1",
		content:      "README contents",
	})
	if len(agent.session.Messages) != 0 {
		t.Fatalf("session messages length = %d, want 0 for headless history-only helper", len(agent.session.Messages))
	}
}

func newToolResultRetentionTestAgent(t *testing.T) *Agent {
	t.Helper()
	var out bytes.Buffer
	return &Agent{
		CurrentModel: "gpt-5.4",
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(nil, &out, &out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-5.4"),
		},
	}
}

func retentionTestToolCall(id, path string) *tools.ToolCall {
	return &tools.ToolCall{
		ID:      id,
		Tool:    "read_file",
		Args:    map[string]string{"path": path},
		RawArgs: map[string]any{"path": path},
	}
}

type toolResultAssertion struct {
	historyIndex              int
	sessionToolExecutionIndex int
	sessionConversationIndex  int
	toolCallID                string
	content                   string
}

func assertRetainedToolResult(t *testing.T, agent *Agent, assertion toolResultAssertion) {
	t.Helper()
	assertToolResultHistory(t, agent, assertion)
	assertSessionToolExecution(t, agent, assertion)
	assertSessionConversationToolResult(t, agent, assertion)
}

func assertToolResultHistory(t *testing.T, agent *Agent, assertion toolResultAssertion) {
	t.Helper()
	if len(agent.History) <= assertion.historyIndex {
		t.Fatalf("History length = %d, want index %d", len(agent.History), assertion.historyIndex)
	}
	historyMsg := agent.History[assertion.historyIndex]
	if historyMsg.Role != "tool" || historyMsg.ToolCallID != assertion.toolCallID || historyMsg.ToolName != "read_file" || historyMsg.Content != assertion.content {
		t.Fatalf("History[%d] = %#v, want retained tool result", assertion.historyIndex, historyMsg)
	}
}

func assertSessionToolExecution(t *testing.T, agent *Agent, assertion toolResultAssertion) {
	t.Helper()
	if len(agent.session.Messages) <= assertion.sessionToolExecutionIndex {
		t.Fatalf("session messages length = %d, want tool execution index %d", len(agent.session.Messages), assertion.sessionToolExecutionIndex)
	}
	executionEntry := agent.session.Messages[assertion.sessionToolExecutionIndex]
	if executionEntry.EntryType != "tool_execution" || executionEntry.ToolExecution == nil {
		t.Fatalf("session.Messages[%d] = %#v, want tool execution entry", assertion.sessionToolExecutionIndex, executionEntry)
	}
	if executionEntry.ToolExecution.Name != "read_file" || executionEntry.ToolExecution.ResultPreview != assertion.content || !executionEntry.ToolExecution.Success {
		t.Fatalf("tool execution entry = %#v, want retained read_file success", executionEntry.ToolExecution)
	}
}

func assertSessionConversationToolResult(t *testing.T, agent *Agent, assertion toolResultAssertion) {
	t.Helper()
	if len(agent.session.Messages) <= assertion.sessionConversationIndex {
		t.Fatalf("session messages length = %d, want conversation index %d", len(agent.session.Messages), assertion.sessionConversationIndex)
	}
	conversationEntry := agent.session.Messages[assertion.sessionConversationIndex]
	if conversationEntry.EntryType != "" || conversationEntry.Role != "tool" || conversationEntry.ToolCallID != assertion.toolCallID || conversationEntry.Content != assertion.content {
		t.Fatalf("session.Messages[%d] = %#v, want retained conversation tool result", assertion.sessionConversationIndex, conversationEntry)
	}
}

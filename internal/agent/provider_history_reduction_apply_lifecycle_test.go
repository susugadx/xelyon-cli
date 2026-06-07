package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestProviderHistoryReductionApplyDoesNotMutateRawHistoryOrSession(t *testing.T) {
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_old", Path: "README.md", StartLine: 1, EndLine: 2},
	)
	rawReadResult := strings.Repeat("raw read result\n", 240)
	assistant := providerHistoryAssistantToolCall("call_old", "read_file")
	toolResult := providerHistoryToolResult("call_old", "read_file", rawReadResult)
	agent := &Agent{
		CurrentModel: "test-model",
		Runtime:      &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			assistant,
			toolResult,
			{Role: "assistant", Content: "after raw read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest"),
			{Role: "assistant", Content: "done"},
		},
	}
	agent.session = history.NewSession("test-model")
	agent.session.AddMessageFromAPI(assistant, "test-model")
	agent.session.AddMessageFromAPI(toolResult, "test-model")
	agent.session.AddToolExecution("read_file", map[string]string{"path": "README.md"}, rawReadResult, true, "test-model")

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if result.History[1].Content == "raw read result" || !strings.HasPrefix(result.History[1].Content, "[omitted old read_file result;") {
		t.Fatalf("projection tool content = %q, want replacement placeholder", result.History[1].Content)
	}
	result.History[1].Content = "provider mutated projection"

	if agent.History[1].Content != rawReadResult {
		t.Fatalf("Agent.History[1].Content = %q, want raw read result", agent.History[1].Content)
	}
	if agent.session.Messages[1].Content != rawReadResult {
		t.Fatalf("session conversation tool content = %q, want raw read result", agent.session.Messages[1].Content)
	}
	assertProviderHistoryToolExecutionPreviewPreservesRaw(t, agent.session.Messages[2].ToolExecution, "read_file", rawReadResult)
}

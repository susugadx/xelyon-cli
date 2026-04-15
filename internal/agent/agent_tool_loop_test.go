package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestShouldAbortToolLoop_SameToolRepeated(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)

	cfg := agent.cfg()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
	}

	count := 0

	abort := agent.shouldAbortToolLoop(toolCall, nil, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort on first call")
	}
	if count != 1 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 1", count)
	}

	abort = agent.shouldAbortToolLoop(toolCall, toolCall, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort on second call")
	}
	if count != 2 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 2", count)
	}

	abort = agent.shouldAbortToolLoop(toolCall, toolCall, &count)
	if !abort {
		t.Error("shouldAbortToolLoop() should abort on third call (threshold reached)")
	}
	if count != 3 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 3", count)
	}
	if len(agent.History) == 0 {
		t.Error("shouldAbortToolLoop() should add warning message to History")
	}
}

func TestShouldAbortToolLoop_DifferentTools(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)

	cfg := agent.cfg()
	originalThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 3
	defer func() {
		cfg.LoopDetection.Threshold = originalThreshold
	}()

	toolCall1 := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
	}

	toolCall2 := &tools.ToolCall{
		Tool: "write_file",
		Args: map[string]string{"path": "/test.txt"},
	}

	count := 0

	abort := agent.shouldAbortToolLoop(toolCall1, nil, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort")
	}

	abort = agent.shouldAbortToolLoop(toolCall2, toolCall1, &count)
	if abort {
		t.Error("shouldAbortToolLoop() should not abort when tool changes")
	}
	if count != 1 {
		t.Errorf("shouldAbortToolLoop() count = %d, want 1 (should reset)", count)
	}
}

func TestExecuteToolCall_UpdatesHistory(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)

	initialHistoryLen := len(agent.History)
	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/nonexistent.txt"),
	}

	agent.executeToolCall("test response", toolCall)

	if len(agent.History) != initialHistoryLen+2 {
		t.Errorf("executeToolCall() added %d messages, want 2 (assistant + tool result)", len(agent.History)-initialHistoryLen)
	}
	if agent.History[len(agent.History)-2].Role != "assistant" {
		t.Errorf("executeToolCall() added message with role = %v, want 'assistant'", agent.History[len(agent.History)-2].Role)
	}
}

func TestExecuteToolCall_UpdatesStats(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/test.txt"),
	}

	agent.executeToolCall("test response", toolCall)

	if agent.Stats.AssistantMessages != 1 {
		t.Errorf("executeToolCall() AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}
	if agent.Stats.ToolExecutions["read_file"] != 1 {
		t.Errorf("executeToolCall() ToolExecutions['read_file'] = %d, want 1", agent.Stats.ToolExecutions["read_file"])
	}
}

func TestExecuteToolCallWithResult(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCall := &tools.ToolCall{
		Tool: "read_file",
		Args: testReadFileArgs("/nonexistent-file-for-test.txt"),
	}

	result := agent.executeToolCallWithResult("test response", toolCall)
	if result == "" {
		t.Error("executeToolCallWithResult() should return non-empty result")
	}
	if !containsAnyNeedle(result, []string{"Error", "error", "not found", "Security"}) {
		t.Logf("Result: %s", result)
	}
}

func containsAnyNeedle(s string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

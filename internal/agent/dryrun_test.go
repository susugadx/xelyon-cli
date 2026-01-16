package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// MockProvider for testing
type mockDryRunProvider struct {
	api.Provider // Embed interface
}

func (m *mockDryRunProvider) Name() string {
	return "mock"
}

func TestAgent_DryRunMode(t *testing.T) {
	// Initialize agent with dry run mode enabled
	agent := NewAgent("test-model", &mockDryRunProvider{})
	agent.DryRunMode = true

	// Create a tool call that would normally modify a file
	toolCall := &tools.ToolCall{
		Tool: "write_file",
		Args: map[string]string{
			"path":    "test_file.txt",
			"content": "Hello World",
		},
	}

	// Execute tool call
	// In dry run mode, this should NOT create the file and should NOT return a change
	agent.executeToolCall("I will write to the file.", toolCall)

	// Check if change stack is empty (no changes should be recorded)
	if len(agent.changeStack) > 0 {
		t.Errorf("Expected change stack to be empty in dry run mode, got %d changes", len(agent.changeStack))
	}

	// Check history for dry run message
	lastMsg := agent.History[len(agent.History)-1]
	if lastMsg.Role != "user" { // Tool result is stored as user message
		t.Errorf("Expected last message role to be user, got %s", lastMsg.Role)
	}

	// Check content contains Dry Run indicator
	expectedContent := "[Dry Run] Tool execution simulated"
	if !strings.Contains(lastMsg.Content, expectedContent) {
		t.Errorf("Expected history to contain '%s', got '%s'", expectedContent, lastMsg.Content)
	}
}

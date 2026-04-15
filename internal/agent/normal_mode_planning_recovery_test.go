package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestNormalMode_MixedToolResponse_CountsAssistantOnce(t *testing.T) {
	testFile := t.TempDir() + "/sample.txt"
	if err := os.WriteFile(testFile, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			fmt.Sprintf("I'll inspect the file first.\n{\"tool\": \"read_file\", \"args\": {\"paths\": [%q]}}", testFile),
			"Done.",
		},
	}

	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = NewSessionStats("test")
	agent.setAutoApprove(true)

	if err := agent.runNormalMode(context.Background(), "inspect file", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}

	if agent.Stats.AssistantMessages != 2 {
		t.Fatalf("AssistantMessages = %d, want 2 (one mixed tool turn + one final response)", agent.Stats.AssistantMessages)
	}
}

func TestNormalMode_PlanJSONFallback(t *testing.T) {
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return `Here is my plan:
{"plan": {"summary": "Test plan", "steps": [{"id": 1, "description": "Do step 1", "tools": ["bash"]}]}}`, nil
			}
			return "OK, I'll execute it directly.", nil
		},
	}

	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "do something", nil); err != nil {
		t.Fatalf("runNormalMode() returned error: %v", err)
	}

	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; !strings.Contains(got, "Do NOT output JSON directly") {
		t.Fatalf("expected direct execution recovery prompt, got %q", got)
	}
	for _, msg := range agent.History {
		if strings.Contains(msg.Content, "Switching to step-by-step") {
			t.Fatalf("step execution fallback should be removed, got %#v", agent.History)
		}
	}
}

func TestNormalMode_PlanJSONParseFailed(t *testing.T) {
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return `{"plan": {"summary": "broken plan"}}`, nil
			}
			return "OK, I'll just do it directly. Task completed.", nil
		},
	}

	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "do something", nil); err != nil {
		t.Fatalf("runNormalMode() returned error: %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; !strings.Contains(got, "Do NOT output JSON directly") {
		t.Fatalf("expected direct execution recovery prompt, got %q", got)
	}
}

func TestNormalMode_TextPlanHardFallback_DoesNotDuplicateAssistantHistory(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = "phase"

	responses := []string{
		"Plan try 1\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Plan try 2\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Plan try 3\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Plan try 4\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Plan try 5\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Plan try 6\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
	}
	provider := &sequenceMockProvider{
		name:      "test",
		responses: responses,
	}

	agent := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: provider,
		Runtime: &AgentRuntime{
			Config: cfg,
			UI:     ui.NewRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{}),
		},
		History: []api.Message{},
	}
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "do something", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}

	lastResponse := responses[len(responses)-1]
	assistantCount := 0
	for _, msg := range agent.History {
		if msg.Role == "assistant" && msg.Content == lastResponse {
			assistantCount++
		}
	}
	if assistantCount != 1 {
		t.Fatalf("assistantCount for final fallback response = %d, want 1", assistantCount)
	}
	if agent.Stats.AssistantMessages != 0 {
		t.Fatalf("AssistantMessages = %d, want 0 for fallback-only path", agent.Stats.AssistantMessages)
	}
}

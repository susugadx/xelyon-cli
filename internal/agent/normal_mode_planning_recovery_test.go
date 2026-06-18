package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const normalModeDirectExecutionPromptFragment = "Do NOT output JSON directly"

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
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; !strings.Contains(got, normalModeDirectExecutionPromptFragment) {
		t.Fatalf("expected direct execution recovery prompt, got %q", got)
	}
	for _, msg := range agent.History {
		if strings.Contains(msg.Content, "Switching to step-by-step") {
			t.Fatalf("step execution fallback should be removed, got %#v", agent.History)
		}
	}
}

func TestNormalMode_PlanJSONFallback_IgnoresNoStepPlanJSON(t *testing.T) {
	response := `{"plan":{"summary":"Already done","findings":["Existing behavior is enough"],"evidence":["README.md"],"constraints":["Do not edit"],"steps":[]}}`
	agent, provider := runNormalModeWithSingleResponse(t, "do something", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"summary":"Already done"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresIDOnlyStepPlanJSON(t *testing.T) {
	response := `{"plan":{"summary":"Fix","steps":[{"id":1}]}}`
	agent, provider := runNormalModeWithSingleResponse(t, "return plan-like json", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"steps":[{"id":1}]`)
}

func TestNormalMode_PlanJSONFallback_DetectsSchemaInvalidPlanJSON(t *testing.T) {
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			if call == 0 {
				return `{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}`, nil
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
	if got := capturedHistories[1][len(capturedHistories[1])-1].Content; !strings.Contains(got, normalModeDirectExecutionPromptFragment) {
		t.Fatalf("expected direct execution recovery prompt, got %q", got)
	}
}

func TestNormalMode_PlanJSONFallback_IgnoresUnrelatedStepsJSON(t *testing.T) {
	agent, provider := runNormalModeWithSingleResponse(t, "return recipe json", `{"title":"recipe","steps":["mix","bake"]}`)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"title":"recipe"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresUnrelatedObjectStepsJSON(t *testing.T) {
	response := `{"title":"recipe","steps":[{"id":1,"description":"mix"}]}`
	agent, provider := runNormalModeWithSingleResponse(t, "return recipe json", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"description":"mix"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresLegacyShapeWithoutWrapper(t *testing.T) {
	response := `{"summary":"recipe","steps":[{"id":1,"description":"mix","tools":["bowl"]}]}`
	agent, provider := runNormalModeWithSingleResponse(t, "return recipe json", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"summary":"recipe"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresLegacyRetrySchemaWithoutWrapper(t *testing.T) {
	response := `{"title":"recipe","goal":"dessert","assumptions":["oven"],"steps":[{"id":1,"description":"mix","expected_output":"batter"}]}`
	agent, provider := runNormalModeWithSingleResponse(t, "return recipe json", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"expected_output":"batter"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresUnrelatedPlanFieldJSON(t *testing.T) {
	agent, provider := runNormalModeWithSingleResponse(t, "return subscription json", `{"title":"monthly","plan":"free"}`)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"plan":"free"`)
}

func TestNormalMode_PlanJSONFallback_IgnoresToolCallJSONWithPlanShapedSteps(t *testing.T) {
	response := "I'll show the tool call shape:\n```json\n" +
		`{"tool":"read_file","steps":[{"id":1,"description":"Read parser","tools":["read_file"]}],"args":{"paths":["internal/agent/plan/parser.go"]}}` +
		"\n```"
	agent, provider := runNormalModeWithSingleResponse(t, "return tool example", response)

	assertNormalModeDidNotRequestPlanRecovery(t, agent, provider, `"tool":"read_file"`)
}

func runNormalModeWithSingleResponse(t *testing.T, userMessage string, response string) (*Agent, *scriptedChatProvider) {
	t.Helper()

	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			return response, nil
		},
	}

	agent := newAgentChatTestAgent(t, provider)
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), userMessage, nil); err != nil {
		t.Fatalf("runNormalMode() returned error: %v", err)
	}

	return agent, provider
}

func assertNormalModeDidNotRequestPlanRecovery(t *testing.T, agent *Agent, provider *scriptedChatProvider, originalResponseFragment string) {
	t.Helper()

	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1 without plan JSON recovery", provider.callCount)
	}
	for _, msg := range agent.History {
		if strings.Contains(msg.Content, normalModeDirectExecutionPromptFragment) {
			t.Fatalf("normal mode should not append plan recovery prompt, got %#v", agent.History)
		}
	}
	if got := agent.History[len(agent.History)-1].Content; !strings.Contains(got, originalResponseFragment) {
		t.Fatalf("last history message = %q, want original response fragment %q", got, originalResponseFragment)
	}
}

func TestNormalMode_TextPlanHardFallback_DoesNotDuplicateAssistantHistory(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = "phase"

	responses := []string{
		"Here is the plan (try 1):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Here is the plan (try 2):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Here is the plan (try 3):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Here is the plan (try 4):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Here is the plan (try 5):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
		"Here is the plan (try 6):\n1. create file\n2. update config\n3. run test\n4. fix error\n5. summarize result",
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
			UI:     uiruntime.NewRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{}),
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

package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type commentSignalTool struct{}

type failingWriteTool struct{}

func (t *commentSignalTool) Name() string { return "comment_signal" }

func (t *commentSignalTool) Description() string { return "Returns a comment signal for testing." }

func (t *commentSignalTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"note": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (t *commentSignalTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	return "[COMMENT] " + args["note"], nil, nil
}

func (t *failingWriteTool) Name() string { return "write_file" }

func (t *failingWriteTool) Description() string { return "Returns a write-like failure for testing." }

func (t *failingWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type": "string",
			},
			"content": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (t *failingWriteTool) Run(_ tools.ExecutionContext, _ map[string]string) (string, *tools.FileChange, error) {
	return "exit status 1", nil, nil
}

func newTurnRunnerTestAgent(provider api.Provider, cfg *config.Config, promptInput string, out *bytes.Buffer, extraTools ...tools.Tool) *Agent {
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(promptInput), out, out)
	registry := tools.DefaultRegistry.Clone()
	for _, tool := range extraTools {
		registry.Register(tool)
	}
	runtime.Registry = registry

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	agent.setAutoApprove(true)
	return agent
}

func newForcedHardRetryState(errOutput string) *retryState {
	return &retryState{
		count:       3,
		lastErrorFP: errorFingerprint(errOutput),
		sameCount:   stalledRetryThreshold,
		stalledRuns: stalledHardThreshold,
	}
}

func TestRunNormalMode_CompletionHookFailureRetries(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase
	cfg.Hooks.OnCompletion = []string{"exit 1"}
	cfg.Hooks.Timeout = 1
	cfg.Hooks.MaxRetry = 2

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			"変更が完了しました。",
			"修正が完了しました。",
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out)
	agent.agentWorkspaceState.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "finish it", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if !strings.Contains(out.String(), "Completion hook failed (1/2). Asking AI to fix...") {
		t.Fatalf("expected retry output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Hook retry limit reached (2/2). Proceeding with completion.") {
		t.Fatalf("expected hook retry limit output, got %q", out.String())
	}

	foundFeedback := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "Hook command \"exit 1\" failed") {
			foundFeedback = true
			break
		}
	}
	if !foundFeedback {
		t.Fatalf("expected hook feedback to be appended to history, got %#v", agent.History)
	}
}

func TestRunNormalMode_CommentFlowRequestsReplan(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"comment_signal","args":{"note":"Use search_code before editing."}}`,
			"別案で進めます。",
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &commentSignalTool{})
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "do it", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}

	foundFeedback := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "The previous tool execution was NOT performed because the user selected comment") {
			foundFeedback = true
			break
		}
	}
	if !foundFeedback {
		t.Fatalf("expected comment feedback in history, got %#v", agent.History)
	}
}

func TestExecuteStepV2_SelectorRetryRestartsStep(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"write_file","args":{"path":"retry.txt","content":"x"}}`,
			"Retry path completed.",
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "1\n", &out, &failingWriteTool{})

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Retry this step", Status: "pending", Tools: []string{"bash"}},
		},
	}

	if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, newForcedHardRetryState("exit status 1")); err != nil {
		t.Fatalf("executeStepV2() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if !strings.Contains(out.String(), "✓ Retry") || !strings.Contains(out.String(), "✓ Step 1 completed") {
		t.Fatalf("expected retry selector flow output, got %q", out.String())
	}
}

func TestExecuteStepV2_SelectorCommentRestartsStepWithInstructions(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"write_file","args":{"path":"comment.txt","content":"x"}}`,
			"Comment path completed.",
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "2\nUse search first\n\n\n", &out, &failingWriteTool{})

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Comment this step", Status: "pending", Tools: []string{"bash"}},
		},
	}

	if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, newForcedHardRetryState("exit status 1")); err != nil {
		t.Fatalf("executeStepV2() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}

	foundComment := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "Use search first") {
			foundComment = true
			break
		}
	}
	if !foundComment {
		t.Fatalf("expected manual comment instructions in history, got %#v", agent.History)
	}
}

func TestExecuteStepV2_SelectorSkipSkipsStep(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"write_file","args":{"path":"skip.txt","content":"x"}}`,
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "3\n", &out, &failingWriteTool{})

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Skip this step", Status: "pending", Tools: []string{"bash"}},
		},
	}

	if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, newForcedHardRetryState("exit status 1")); err != nil {
		t.Fatalf("executeStepV2() error = %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider.callCount = %d, want 1", provider.callCount)
	}
	if !strings.Contains(out.String(), "⏭️  Step 1 skipped by user") {
		t.Fatalf("expected skip output, got %q", out.String())
	}
}

func TestExecuteStepV2_SoftStallRetriesWithStrategyChange(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"write_file","args":{"path":"soft.txt","content":"x"}}`,
			"Strategy change completed.",
		},
	}
	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &failingWriteTool{})

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Recover with strategy change", Status: "pending", Tools: []string{"bash"}},
		},
	}

	rs := &retryState{
		count:       2,
		lastErrorFP: errorFingerprint("exit status 1"),
		sameCount:   stalledRetryThreshold - 1,
	}

	if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, rs); err != nil {
		t.Fatalf("executeStepV2() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if !strings.Contains(out.String(), "Retrying with strategy change") {
		t.Fatalf("expected strategy-change retry output, got %q", out.String())
	}

	foundMessage := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "A similar failure has now occurred 3 times in a row") {
			foundMessage = true
			break
		}
	}
	if !foundMessage {
		t.Fatalf("expected strategy-change retry message in history, got %#v", agent.History)
	}
}

package agent

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type commentSignalTool struct{}

type failingWriteTool struct{}
type finalCheckWriteTool struct{}

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

func (t *finalCheckWriteTool) Name() string { return "final_check_write" }

func (t *finalCheckWriteTool) Description() string {
	return "Writes file content for final check retry tests."
}

func (t *finalCheckWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string"},
			"content": map[string]interface{}{"type": "string"},
		},
	}
}

func (t *finalCheckWriteTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	path := args["path"]
	content := args["content"]
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", nil, err
	}
	return "final check retry wrote file", &tools.FileChange{
		FilePath: path,
		Tool:     "final_check_write",
		Details: []tools.FileChangeDetail{{
			FilePath: path,
			Action:   "modified",
		}},
	}, nil
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
	if len(extraTools) > 0 {
		// Characterization tests inject explicit doubles such as failing write tools.
		// They should exercise those doubles directly instead of the runtime surface policy.
		agent.registry().ClearExcludedTools()
	}
	return agent
}

func newForcedHardRetryState(errOutput string) *turnsupport.RetryState {
	state := &turnsupport.RetryState{}
	for i := 0; i < 4; i++ {
		state.RecordFailure(errOutput)
	}
	return state
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func newCommittedGitRepo(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")

	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write repo file: %v", err)
	}

	runGit(t, dir, "add", "main.go")
	runGit(t, dir, "commit", "-m", "initial")
	return dir, file
}

func TestRunNormalMode_FinalCheckFailureRequestsFix(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesPhase

	provider := &sequenceMockProvider{
		name: "test",
	}
	repoDir, filePath := newCommittedGitRepo(t)
	cfg.FinalChecks.Commands = []string{"grep -q fixed " + filePath}
	cfg.FinalChecks.Timeout = 1

	provider.responses = []string{
		`{"tool":"final_check_write","args":{"path":"` + filePath + `","content":"package main\n\nfunc main() { println(\"done\") }\n"}}`,
		"Would you like me to continue?",
		`{"tool":"final_check_write","args":{"path":"` + filePath + `","content":"package main\n\nfunc main() { println(\"fixed\") }\n"}}`,
		"Would you like me to continue?",
	}

	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &finalCheckWriteTool{})
	chdirForTest(t, repoDir)
	agent.Stats = NewSessionStats("test")

	if err := agent.runNormalMode(context.Background(), "finish it", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}
	if provider.callCount != 4 {
		t.Fatalf("provider.callCount = %d, want 4", provider.callCount)
	}
	if !strings.Contains(out.String(), "Final check command failed. Asking AI to fix...") {
		t.Fatalf("expected final check retry output, got %q", out.String())
	}

	foundFeedback := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "Final check failed. Command") {
			foundFeedback = true
			break
		}
	}
	if !foundFeedback {
		t.Fatalf("expected final check feedback to be appended to history, got %#v", agent.History)
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

func TestHandleNormalModeNoToolResponse_StalledHardDelegatesBackToAI(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out, &failingWriteTool{})
	runner := newTurnRunner(agent, context.Background())
	rs := newForcedHardRetryState("exit status 1")

	response := `{"tool":"write_file","args":{"path":"retry.txt","content":"x"}}`
	toolCalls := runner.prepareToolCalls(response)
	if err := runner.processNormalModeToolCalls(response, toolCalls, &normalModeState{}, rs); err != nil {
		t.Fatalf("processNormalModeToolCalls() error = %v", err)
	}
	if rs.Count() != 0 || rs.SameCount() != 0 {
		t.Fatalf("retry state not reset after stalledHard: %+v", rs)
	}
	if !strings.Contains(out.String(), "Could not complete the task automatically. Letting AI respond...") {
		t.Fatalf("expected stalledHard delegation output, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_TextPlanFirstRedirect(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}

	response := `Here is the plan:
1. Create test file
2. Fix lint errors
3. Update README
4. Implement handler
5. Run tests`
	action := runner.handleNormalModeNoToolResponse(response, cfg, state)
	if action != normalModeContinue {
		t.Fatalf("action = %v, want normalModeContinue", action)
	}
	if state.textPlanRedirectCount != 1 {
		t.Fatalf("textPlanRedirectCount = %d, want 1", state.textPlanRedirectCount)
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "assistant" || last.Content != response {
		t.Fatalf("expected assistant response only in history, got %#v", last)
	}
	if len(agent.runtimeDirectives) != 1 || !strings.Contains(agent.runtimeDirectives[0], "do not respond with another numbered text plan") {
		t.Fatalf("expected first redirect runtime directive, got %#v", agent.runtimeDirectives)
	}
}

func TestHandleNormalModeNoToolResponse_TextPlanForcesDirectExecution(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{textPlanRedirectCount: maxTextPlanRedirects}

	response := `Plan:
1. Create test file
2. Fix lint errors
3. Update README
4. Implement handler
5. Run tests`
	action := runner.handleNormalModeNoToolResponse(response, cfg, state)
	if action != normalModeContinue {
		t.Fatalf("action = %v, want normalModeContinue", action)
	}
	if state.textPlanRedirectCount != maxTextPlanRedirects+1 {
		t.Fatalf("textPlanRedirectCount = %d, want %d", state.textPlanRedirectCount, maxTextPlanRedirects+1)
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "assistant" || last.Content != response {
		t.Fatalf("expected assistant response only in history, got %#v", last)
	}
	if len(agent.runtimeDirectives) != 1 || !strings.Contains(agent.runtimeDirectives[0], "previous response was still a text plan") {
		t.Fatalf("expected forced execution runtime directive, got %#v", agent.runtimeDirectives)
	}
}

func TestHandleStrReplaceErrors_QueuesRuntimeDirectiveWithoutFakeSystemMessage(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.LoopDetection.Threshold = 2
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	toolCall := &tools.ToolCall{ID: "call_replace", Tool: "str_replace"}
	result := "Error: old_str not found in file.go"

	if handled := agent.handleStrReplaceErrors(toolCall, result); handled {
		t.Fatal("first str_replace failure should not trigger loop handling")
	}
	if handled := agent.handleStrReplaceErrors(toolCall, result); !handled {
		t.Fatal("second str_replace failure should trigger loop handling")
	}
	if len(agent.History) != 1 || agent.History[0].Role != "tool" || agent.History[0].ToolCallID != "call_replace" {
		t.Fatalf("history = %#v, want one tool result data message", agent.History)
	}
	if strings.Contains(agent.History[0].Content, "[SYSTEM") {
		t.Fatalf("str_replace failure data should not contain fake system marker: %q", agent.History[0].Content)
	}
	if len(agent.runtimeDirectives) != 1 || agent.runtimeDirectives[0] != strReplaceLoopRuntimeDirective {
		t.Fatalf("runtime directives = %#v, want str_replace loop directive", agent.runtimeDirectives)
	}
	prompt := agent.normalModeSystemPromptForRequest(context.Background(), "retry edit", false)
	if !strings.Contains(prompt, strReplaceLoopRuntimeDirective) {
		t.Fatalf("system prompt = %q, want str_replace loop runtime directive", prompt)
	}
	if len(agent.runtimeDirectives) != 1 || agent.runtimeDirectives[0] != strReplaceLoopRuntimeDirective {
		t.Fatalf("runtime directives should stay pending until provider delivery succeeds, got %#v", agent.runtimeDirectives)
	}
}

func TestHandleNormalModeNoToolResponse_TextPlanFallsBackToFinalResponse(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{textPlanRedirectCount: maxTextPlanHardLimit}

	response := `Here is the plan:
1. Create test file
2. Fix lint errors
3. Update README
4. Implement handler
5. Run tests`
	action := runner.handleNormalModeNoToolResponse(response, cfg, state)
	if action != normalModeBreak {
		t.Fatalf("action = %v, want normalModeBreak", action)
	}
	if state.fallbackResponse != response {
		t.Fatalf("fallbackResponse = %q, want %q", state.fallbackResponse, response)
	}
	if !strings.Contains(out.String(), "Returning response to user.") {
		t.Fatalf("expected hard fallback output, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_NumberedSummaryWithoutStrongPlanSignalDoesNotRecover(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}

	response := `Done.
1. Updated foo.go
2. Added tests
3. Ran go test`
	action := runner.handleNormalModeNoToolResponse(response, cfg, state)
	if action != normalModeDone {
		t.Fatalf("action = %v, want normalModeDone", action)
	}
	if state.textPlanRedirectCount != 0 {
		t.Fatalf("textPlanRedirectCount = %d, want 0", state.textPlanRedirectCount)
	}
	if strings.Contains(out.String(), "Text plan detected") {
		t.Fatalf("expected no text plan recovery output, got %q", out.String())
	}
}

func TestNormalModeToolResultHandler_TracksWriteFailure(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	handler := newNormalModeToolResultHandler(runner, &normalModeState{})

	tc := &tools.ToolCall{
		Tool: "write_file",
		Args: map[string]string{"path": "failure.txt", "content": "x"},
	}

	handler.Handle(tc, toolruntime.Result{Result: "exit status 1", Error: true})

	if got := handler.LastFailedResult(); got != "exit status 1" {
		t.Fatalf("LastFailedResult() = %q, want %q", got, "exit status 1")
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "user" || !strings.Contains(last.Content, "[Tool Result for write_file]") {
		t.Fatalf("expected tool result in history, got %#v", last)
	}
}

package agent

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	lsplib "github.com/susugadx/xelyon-cli/internal/lsp"
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

func setUnexportedField(t *testing.T, target any, fieldName string, value any) {
	t.Helper()

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		t.Fatalf("target must be a non-nil pointer")
	}
	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("field %q not found", fieldName)
	}
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func newFakeLSPClientWithError(t *testing.T, rootDir, filePath, message string) *lsplib.Client {
	t.Helper()

	client := lsplib.NewClient(rootDir)
	server := lsplib.NewServer("go")
	fileURI := lsplib.FileToURI(filePath)

	setUnexportedField(t, server, "initialized", true)
	setUnexportedField(t, server, "openDocs", map[string]struct{}{fileURI: {}})
	setUnexportedField(t, server, "diagnostics", map[string][]lsplib.Diagnostic{
		fileURI: {
			{
				Range: lsplib.Range{
					Start: lsplib.Position{Line: 0, Character: 0},
					End:   lsplib.Position{Line: 0, Character: 4},
				},
				Severity: lsplib.DiagnosticSeverityError,
				Message:  message,
			},
		},
	})
	setUnexportedField(t, client, "servers", map[string]*lsplib.Server{
		"go": server,
	})

	return client
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

func TestHandleStepNoToolResponse_AutoContinue(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &stepRunState{}
	step := &plan.PlanStep{ID: 1, Description: "Ask a question", Status: "pending"}

	action, err := runner.handleStepNoToolResponse("Should I proceed with the next step?", step, state)
	if err != nil {
		t.Fatalf("handleStepNoToolResponse() error = %v", err)
	}
	if action != stepLoopContinue {
		t.Fatalf("action = %v, want stepLoopContinue", action)
	}
	if state.continueCount != 1 {
		t.Fatalf("continueCount = %d, want 1", state.continueCount)
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "user" || !strings.Contains(last.Content, "[AUTO-CONTINUE] Yes, proceed with the step") {
		t.Fatalf("expected auto-continue message, got %#v", last)
	}
}

func TestHandleStepNoToolResponse_AlreadyApplied(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	step := &plan.PlanStep{ID: 1, Description: "Already applied", Status: "pending"}

	repoDir, filePath := newCommittedGitRepo(t)
	chdirForTest(t, repoDir)

	before := getGitDiffHash()
	if before == "" {
		t.Skip("git diff hash unavailable")
	}
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() { println(\"done\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to modify repo file: %v", err)
	}

	state := &stepRunState{beforeDiffHash: before}
	action, err := runner.handleStepNoToolResponse("変更が完了しました。", step, state)
	if err != nil {
		t.Fatalf("handleStepNoToolResponse() error = %v", err)
	}
	if action != stepLoopDone {
		t.Fatalf("action = %v, want stepLoopDone", action)
	}
	if !strings.Contains(out.String(), "Step 1 completed (already applied)") {
		t.Fatalf("expected already-applied output, got %q", out.String())
	}
}

func TestHandleStepNoToolResponse_WriteToolsWithoutDiffRequestsRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	step := &plan.PlanStep{ID: 1, Description: "No diff after writes", Status: "pending"}

	repoDir, _ := newCommittedGitRepo(t)
	chdirForTest(t, repoDir)

	before := getGitDiffHash()
	if before == "" {
		t.Skip("git diff hash unavailable")
	}

	state := &stepRunState{
		beforeDiffHash: before,
		stepHadWrites:  true,
	}
	action, err := runner.handleStepNoToolResponse("I changed the files.", step, state)
	if err != nil {
		t.Fatalf("handleStepNoToolResponse() error = %v", err)
	}
	if action != stepLoopContinue {
		t.Fatalf("action = %v, want stepLoopContinue", action)
	}
	if state.continueCount != 1 {
		t.Fatalf("continueCount = %d, want 1", state.continueCount)
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "user" || !strings.Contains(last.Content, "write tools but git diff shows no new changes") {
		t.Fatalf("expected no-diff retry feedback, got %#v", last)
	}
}

func TestHandleStepNoToolResponse_CompletionVerificationRequestsContinue(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	step := &plan.PlanStep{ID: 1, Description: "Verify completion", Status: "pending"}

	rootDir := t.TempDir()
	filePath := filepath.Join(rootDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	chdirForTest(t, rootDir)

	agent.lspClient = newFakeLSPClientWithError(t, rootDir, filePath, "unused variable")
	agent.agentWorkspaceState.changeStack = []tools.FileChange{{FilePath: filePath}}

	state := &stepRunState{}
	action, err := runner.handleStepNoToolResponse("変更が完了しました。", step, state)
	if err != nil {
		t.Fatalf("handleStepNoToolResponse() error = %v", err)
	}
	if action != stepLoopContinue {
		t.Fatalf("action = %v, want stepLoopContinue", action)
	}
	if !state.stepCompletionVerified {
		t.Fatal("expected stepCompletionVerified to be set")
	}
	last := agent.History[len(agent.History)-1]
	if last.Role != "user" || !strings.Contains(last.Content, "Completion verification failed") {
		t.Fatalf("expected completion verification feedback, got %#v", last)
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
	if err := runner.processNormalModeToolCalls(response, toolCalls, rs); err != nil {
		t.Fatalf("processNormalModeToolCalls() error = %v", err)
	}
	if rs.count != 0 || rs.sameCount != 0 || rs.stalledRuns != 0 {
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

	response := `1. Create test file
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
	if last.Role != "user" || !strings.Contains(last.Content, "Do NOT output plans as numbered text") {
		t.Fatalf("expected first redirect system feedback, got %#v", last)
	}
}

func TestHandleNormalModeNoToolResponse_TextPlanForcesDirectExecution(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{textPlanRedirectCount: maxTextPlanRedirects}

	response := `1. Create test file
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
	if last.Role != "user" || !strings.Contains(last.Content, "STOP planning. Pick the FIRST change") {
		t.Fatalf("expected forced execution system feedback, got %#v", last)
	}
}

func TestHandleNormalModeNoToolResponse_TextPlanFallsBackToFinalResponse(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{textPlanRedirectCount: maxTextPlanHardLimit}

	response := `1. Create test file
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

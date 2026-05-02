package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestFinalCheckRetryState_StallsOnRepeatedFailureWithoutChanges(t *testing.T) {
	state := &finalCheckRetryState{}
	result := finalCheckRunResult{
		needsContinue:      true,
		feedback:           "[SYSTEM] Final check failed",
		failureFingerprint: "same failure",
	}
	changeFingerprint := "change-a"

	if stalled := state.recordFailure(result, changeFingerprint); stalled {
		t.Fatal("first repeated failure should not stall")
	}
	if stalled := state.recordFailure(result, changeFingerprint); !stalled {
		t.Fatal("second identical failure without changes should stall")
	}
}

func TestFinalCheckRetryState_ResetsWhenChangedFilesAdvance(t *testing.T) {
	state := &finalCheckRetryState{}
	result := finalCheckRunResult{
		needsContinue:      true,
		feedback:           "[SYSTEM] Final check failed",
		failureFingerprint: "same failure",
	}

	if stalled := state.recordFailure(result, "change-a"); stalled {
		t.Fatal("first failure should not stall")
	}
	if stalled := state.recordFailure(result, "change-b"); stalled {
		t.Fatal("changed fingerprint advance should reset no-progress detection")
	}
}

func TestFinalCheckRetryState_DoesNotStallWithoutProgressFingerprint(t *testing.T) {
	state := &finalCheckRetryState{}
	result := finalCheckRunResult{
		needsContinue:      true,
		feedback:           "[SYSTEM] Final check failed",
		failureFingerprint: "same failure",
	}

	if stalled := state.recordFailure(result, ""); stalled {
		t.Fatal("first failure without progress fingerprint should not stall")
	}
	if stalled := state.recordFailure(result, ""); stalled {
		t.Fatal("unknown progress should not be treated as no progress")
	}
}

func TestFinalCheckRetryState_DoesNotStallWhenSameFileContentAdvances(t *testing.T) {
	targetFile := filepath.Join("/tmp", "foo.go")
	agent := &Agent{
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{{
				FilePath: targetFile,
				Tool:     "write_file",
				Details: []tools.FileChangeDetail{{
					FilePath: targetFile,
					Action:   "modified",
				}},
			}},
		},
	}
	firstFingerprint := agent.recordedTaskChangeFingerprint()
	if firstFingerprint == "" {
		t.Fatal("expected non-empty fingerprint after first file change")
	}

	state := &finalCheckRetryState{}
	result := finalCheckRunResult{
		needsContinue:      true,
		feedback:           "[SYSTEM] Final check failed",
		failureFingerprint: "same failure",
	}
	if stalled := state.recordFailure(result, firstFingerprint); stalled {
		t.Fatal("first failure should not stall")
	}

	agent.changeStack = append(agent.changeStack, tools.FileChange{
		FilePath: targetFile,
		Tool:     "write_file",
		Details: []tools.FileChangeDetail{{
			FilePath: targetFile,
			Action:   "modified",
		}},
	})
	secondFingerprint := agent.recordedTaskChangeFingerprint()
	if secondFingerprint == "" {
		t.Fatal("expected non-empty fingerprint after second file change")
	}
	if secondFingerprint == firstFingerprint {
		t.Fatal("expected fingerprint to change when the same file content advances")
	}

	if stalled := state.recordFailure(result, secondFingerprint); stalled {
		t.Fatal("same file content advance should reset no-progress detection")
	}
}

func TestHandleNormalModeNoToolResponse_FinalChecksNoProgressBreaks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)

	runner := newTurnRunner(agent, context.Background())
	state := newMutatedNormalModeState("/src/main.go")
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("first action = %v, want normalModeContinue", action)
	}
	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeBreak {
		t.Fatalf("second action = %v, want normalModeBreak", action)
	}
	if !strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("expected no-progress final checks warning, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_NoMutationSkipsFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
	response := "Done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeDone {
		t.Fatalf("action = %v, want normalModeDone", action)
	}
	if strings.Contains(out.String(), "Final check command failed") {
		t.Fatalf("expected final checks to be skipped without recorded changes, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_ReadOnlyTurnSkipsFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
	response := "I inspected the file and found no additional changes."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeDone {
		t.Fatalf("action = %v, want normalModeDone", action)
	}
	if strings.Contains(out.String(), "Running final checks") {
		t.Fatalf("expected read-only turn to skip final checks, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_FinalChecksAreTriggeredByMutationNotWording(t *testing.T) {
	disableColors(t)

	// completion 文面の違いではなく、mutation の有無だけで final checks 起動を決める。
	responses := []string{
		"Done.",
		"Would you like me to continue?",
		"完了しました。",
		"Updated foo.go and added tests.",
	}

	for _, response := range responses {
		t.Run("mutation_"+normalizeSubtestName(response), func(t *testing.T) {
			var out bytes.Buffer
			cfg := config.DefaultConfig()
			cfg.FinalChecks.Commands = []string{"exit 1"}
			cfg.FinalChecks.Timeout = 10

			agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)

			runner := newTurnRunner(agent, context.Background())
			state := newMutatedNormalModeState("/src/main.go")

			if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
				t.Fatalf("action = %v, want normalModeContinue", action)
			}
			if !strings.Contains(out.String(), "Final check command failed. Asking AI to fix...") {
				t.Fatalf("expected final checks to run when mutation exists, got %q", out.String())
			}
		})

		t.Run("no_mutation_"+normalizeSubtestName(response), func(t *testing.T) {
			var out bytes.Buffer
			cfg := config.DefaultConfig()
			cfg.FinalChecks.Commands = []string{"exit 1"}
			cfg.FinalChecks.Timeout = 10

			agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
			runner := newTurnRunner(agent, context.Background())
			state := &normalModeState{}

			if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeDone {
				t.Fatalf("action = %v, want normalModeDone", action)
			}
			if strings.Contains(out.String(), "Running final checks") {
				t.Fatalf("expected final checks to be skipped when no mutation exists, got %q", out.String())
			}
		})
	}
}

func TestHandleNormalModeNoToolResponse_DoesNotStallWhenSilentFileContentAdvances(t *testing.T) {
	disableColors(t)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	initialContent := []byte("package main\n\nfunc main() { println(\"v1\") }\n")
	targetFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(targetFile, initialContent, 0o644); err != nil {
		t.Fatalf("failed to write initial target file: %v", err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := newMutatedNormalModeState(targetFile)
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("first action = %v, want normalModeContinue", action)
	}
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to advance target file silently: %v", err)
	}
	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("second action = %v, want normalModeContinue because resulting file state advanced", action)
	}
	if strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("did not expect no-progress warning when resulting file state advanced, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_StallsWhenSilentEditReturnsToSameFileState(t *testing.T) {
	disableColors(t)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	initialContent := []byte("package main\n\nfunc main() { println(\"v1\") }\n")
	targetFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(targetFile, initialContent, 0o644); err != nil {
		t.Fatalf("failed to write initial target file: %v", err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := newMutatedNormalModeState(targetFile)
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("first action = %v, want normalModeContinue", action)
	}
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to update target file silently: %v", err)
	}
	if err := os.WriteFile(targetFile, initialContent, 0o644); err != nil {
		t.Fatalf("failed to revert target file silently: %v", err)
	}
	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeBreak {
		t.Fatalf("second action = %v, want normalModeBreak because resulting file state returned to same fingerprint", action)
	}
	if !strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("expected no-progress warning when resulting file state returned to original fingerprint, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_MaxChangeStackOverflowStillRunsFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{turnMutations: newTurnMutationState()}
	handler := newNormalModeToolResultHandler(runner, state)

	totalMutations := config.MaxChangeStack + 5
	for i := 0; i < totalMutations; i++ {
		path := fmt.Sprintf("/tmp/generated_%d.go", i)
		change := &tools.FileChange{
			FilePath: path,
			Tool:     "write_file",
			Details: []tools.FileChangeDetail{{
				FilePath: path,
				Action:   "modified",
			}},
		}
		handler.Handle(&tools.ToolCall{
			Tool: "write_file",
			Args: map[string]string{"path": path, "content": "x"},
		}, toolruntime.Result{Result: "ok", Change: change})
	}

	if len(agent.changeStack) != config.MaxChangeStack {
		t.Fatalf("changeStack len = %d, want %d", len(agent.changeStack), config.MaxChangeStack)
	}
	if state.turnMutations.snapshot().mutationCount != totalMutations {
		t.Fatalf("mutationCount = %d, want %d", state.turnMutations.snapshot().mutationCount, totalMutations)
	}

	if action := runner.handleNormalModeNoToolResponse("Done.", cfg, state); action != normalModeContinue {
		t.Fatalf("action = %v, want normalModeContinue", action)
	}
	if !strings.Contains(out.String(), "Final check command failed. Asking AI to fix...") {
		t.Fatalf("expected final checks to run even after changeStack overflow, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_NumberedCompletionSummaryAfterMutationRunsFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := newMutatedNormalModeState("/src/main.go")
	response := `Done.
1. Updated foo.go
2. Added tests
3. Ran go test`

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("action = %v, want normalModeContinue", action)
	}
	if strings.Contains(out.String(), "Text plan detected") {
		t.Fatalf("did not expect text plan recovery for numbered completion summary, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Final check command failed. Asking AI to fix...") {
		t.Fatalf("expected mutation-based final checks to run, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_StrongPlanSignalWithMutationUsesFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := newMutatedNormalModeState("/src/main.go")
	response := `Here is the plan:
1. Update foo.go
2. Add tests
3. Run go test`

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("action = %v, want normalModeContinue", action)
	}
	if strings.Contains(out.String(), "Text plan detected") {
		t.Fatalf("did not expect text plan recovery when mutation exists, got %q", out.String())
	}
	if !strings.Contains(out.String(), "Final check command failed. Asking AI to fix...") {
		t.Fatalf("expected mutation-based final checks to run, got %q", out.String())
	}
}

func newMutatedNormalModeState(paths ...string) *normalModeState {
	state := &normalModeState{
		turnMutations: newTurnMutationState(),
	}
	if len(paths) == 0 {
		paths = []string{"/src/main.go"}
	}
	for _, path := range paths {
		state.turnMutations.recordFileChange(tools.FileChange{
			FilePath: path,
			Tool:     "write_file",
			Details: []tools.FileChangeDetail{{
				FilePath: path,
				Action:   "modified",
			}},
		})
	}
	return state
}

func normalizeSubtestName(response string) string {
	name := strings.ToLower(strings.TrimSpace(response))
	replacer := strings.NewReplacer(
		" ", "_",
		".", "_",
		"?", "_",
		"。", "_",
		",", "_",
		"!", "_",
		":", "_",
		";", "_",
		"/", "_",
		"'", "_",
		"\"", "_",
	)
	name = replacer.Replace(name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.Trim(name, "_")
	if name == "" {
		return "response"
	}
	return name
}

func TestRunNormalMode_FinalChecksNoProgressReturnsControl(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	provider := &sequenceMockProvider{
		name: "test",
		responses: []string{
			`{"tool":"final_check_write","args":{"path":"` + filepath.Join(t.TempDir(), "main.go") + `","content":"package main\n\nfunc main() {}\n"}}`,
			"The requested changes are done.",
			"The requested changes are done.",
		},
	}

	agent := newTurnRunnerTestAgent(provider, cfg, "", &out, &finalCheckWriteTool{})
	if err := agent.runNormalMode(context.Background(), "finish it", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}
	if provider.callCount != 3 {
		t.Fatalf("provider.callCount = %d, want 3", provider.callCount)
	}
	if !strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("expected no-progress final checks warning, got %q", out.String())
	}
}

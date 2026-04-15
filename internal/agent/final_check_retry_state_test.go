package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
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
	agent.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	agent.taskChangeOffset = 0

	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
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

func TestHandleNormalModeNoToolResponse_NoRecordedChangesSkipsFinalChecks(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeDone {
		t.Fatalf("action = %v, want normalModeDone", action)
	}
	if strings.Contains(out.String(), "Final check command failed") {
		t.Fatalf("expected final checks to be skipped without recorded changes, got %q", out.String())
	}
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
			"The requested changes are done.",
			"The requested changes are done.",
		},
	}

	agent := newTurnRunnerTestAgent(provider, cfg, "", &out)
	agent.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	agent.taskChangeOffset = 0

	if err := agent.runNormalMode(context.Background(), "finish it", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("provider.callCount = %d, want 2", provider.callCount)
	}
	if !strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("expected no-progress final checks warning, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_DoesNotStallWhenApprovedPlanFilesAdvanceSilently(t *testing.T) {
	disableColors(t)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	targetFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v1\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial target file: %v", err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	agent.setPendingApprovedPlanState("Implementation Plan\n1. Update main.go", true, []string{targetFile})
	agent.activeApprovedPlan = agent.PendingApprovedPlan

	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("first action = %v, want normalModeContinue", action)
	}
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to advance target file silently: %v", err)
	}
	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("second action = %v, want normalModeContinue after silent content advance", action)
	}
	if strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("silent file-content advance should not be treated as no progress, got %q", out.String())
	}
}

func TestHandleNormalModeNoToolResponse_DoesNotStallWhenRecordedFilesAdvanceSilently(t *testing.T) {
	disableColors(t)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	targetFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v1\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial target file: %v", err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	agent := newTurnRunnerTestAgent(&sequenceMockProvider{name: "test"}, cfg, "", &out)
	agent.changeStack = []tools.FileChange{{
		FilePath: targetFile,
		Tool:     "write_file",
		Details: []tools.FileChangeDetail{{
			FilePath: targetFile,
			Action:   "modified",
		}},
	}}
	agent.taskChangeOffset = 0

	runner := newTurnRunner(agent, context.Background())
	state := &normalModeState{}
	response := "The requested changes are done."

	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("first action = %v, want normalModeContinue", action)
	}
	if err := os.WriteFile(targetFile, []byte("package main\n\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatalf("failed to advance target file silently: %v", err)
	}
	if action := runner.handleNormalModeNoToolResponse(response, cfg, state); action != normalModeContinue {
		t.Fatalf("second action = %v, want normalModeContinue after silent content advance", action)
	}
	if strings.Contains(out.String(), "without any task progress") {
		t.Fatalf("silent file-content advance should not be treated as no progress, got %q", out.String())
	}
}

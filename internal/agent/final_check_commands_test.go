package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/finalcheck"
)

func TestRunFinalCheckCommands_NoCommands(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = nil

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go"})
	if result.NeedsContinue {
		t.Error("expected needsContinue=false when no final checks are configured")
	}
	if result.Feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.Feedback)
	}
	if len(result.Checks) != 0 {
		t.Fatalf("Checks = %+v, want empty", result.Checks)
	}
}

func TestRunFinalCheckCommands_SuccessfulCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"echo 'test passed'"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go"})
	if result.NeedsContinue {
		t.Error("expected needsContinue=false for successful command")
	}
	if result.Feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.Feedback)
	}
	if len(result.Checks) != 1 || result.Checks[0].Command != "echo 'test passed'" || result.Checks[0].ExitCode != 0 || result.Checks[0].Status != "passed" {
		t.Fatalf("Checks = %+v, want passed echo command", result.Checks)
	}
}

func TestRunFinalCheckCommands_FailedCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go"})
	if !result.NeedsContinue {
		t.Error("expected needsContinue=true for failed command")
	}
	if !strings.Contains(result.Feedback, "Final check failed") {
		t.Errorf("expected final check feedback, got %q", result.Feedback)
	}
	if strings.Contains(result.Feedback, "[SYSTEM") {
		t.Errorf("expected data-only feedback without fake system marker, got %q", result.Feedback)
	}
	if !strings.Contains(result.Feedback, "exit code 1") {
		t.Errorf("expected exit code in feedback, got %q", result.Feedback)
	}
	if result.FailureFingerprint == "" {
		t.Error("expected non-empty failure fingerprint")
	}
	if len(result.Checks) != 1 || result.Checks[0].Command != "exit 1" || result.Checks[0].ExitCode != 1 || result.Checks[0].Status != "failed" {
		t.Fatalf("Checks = %+v, want failed exit command", result.Checks)
	}
}

func TestRunFinalCheckCommands_ChangedFilesEnv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{`test -n "$XELYON_CHANGED_FILES"`}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go", "/src/util.go"})
	if result.NeedsContinue {
		t.Error("expected needsContinue=false, XELYON_CHANGED_FILES should be set")
	}
}

func TestRunFinalCheckCommands_MultipleCommands_StopsOnFirstFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{
		"echo 'first ok'",
		"exit 42",
		"echo 'should not run'",
	}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go"})
	if !result.NeedsContinue {
		t.Error("expected needsContinue=true when second command fails")
	}
	if !strings.Contains(result.Feedback, "exit code 42") {
		t.Errorf("expected exit code 42 in feedback, got %q", result.Feedback)
	}
	if len(result.Checks) != 2 {
		t.Fatalf("Checks len = %d, want 2: %+v", len(result.Checks), result.Checks)
	}
	if result.Checks[0].Status != "passed" || result.Checks[1].Status != "failed" || result.Checks[1].ExitCode != 42 {
		t.Fatalf("Checks = %+v, want passed then failed exit 42", result.Checks)
	}
}

func TestRunFinalCheckCommands_RecordsLedgerWithoutHistory(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{
		"printf 'first ok'",
		"printf 'second fail' && exit 3",
	}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"src/main.go"})
	if !result.NeedsContinue {
		t.Fatal("expected failed final check to request continuation")
	}
	if len(a.History) != 0 {
		t.Fatalf("History len = %d, want 0", len(a.History))
	}

	snapshot := a.Runtime.TaskLedger.Snapshot()
	passed := snapshot.LastPassedTests.Results()
	if len(passed) != 1 || passed[0].Command() != "printf 'first ok'" || passed[0].Status() != "passed" {
		t.Fatalf("LastPassedTests = %#v", passed)
	}
	failed := snapshot.LastFailedTests.Results()
	if len(failed) != 1 || failed[0].Command() != "printf 'second fail' && exit 3" || failed[0].ExitCode() != 3 {
		t.Fatalf("LastFailedTests = %#v", failed)
	}
	if !strings.Contains(failed[0].Output(), "second fail") {
		t.Fatalf("failed excerpt = %q, want command output", failed[0].Output())
	}
}

func TestRunFinalCheckCommands_Timeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"sleep 30"}
	cfg.FinalChecks.Timeout = 1

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands(context.Background(), []string{"/src/main.go"})
	if !result.NeedsContinue {
		t.Error("expected needsContinue=true for timed-out command")
	}
	if result.Cancelled {
		t.Fatalf("Cancelled = true, want false for per-command timeout")
	}
	if result.Feedback == "" {
		t.Error("expected non-empty feedback for timed-out command")
	}
}

func TestRunFinalCheckCommands_ParentCancelStopsRunningCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	startedFile := filepath.Join(t.TempDir(), "started")
	cfg.FinalChecks.Commands = []string{
		fmt.Sprintf("touch %q; sleep 30", startedFile),
	}
	cfg.FinalChecks.Timeout = 60

	a := newCompletionTestAgent(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan finalcheck.RunResult, 1)
	go func() {
		resultCh <- a.runFinalCheckCommands(ctx, []string{"/src/main.go"})
	}()

	waitForFile(t, startedFile)
	cancel()

	select {
	case result := <-resultCh:
		if !result.Cancelled {
			t.Fatalf("Cancelled = false, want true: %+v", result)
		}
		if !errors.Is(result.Err, context.Canceled) {
			t.Fatalf("Err = %v, want context.Canceled", result.Err)
		}
		if result.NeedsContinue {
			t.Fatal("NeedsContinue = true, want false for parent cancellation")
		}
		if len(result.Checks) != 1 || result.Checks[0].Command != cfg.FinalChecks.Commands[0] || result.Checks[0].ExitCode != -1 || result.Checks[0].Status != "failed" {
			t.Fatalf("Checks = %+v, want cancelled running command as failed/-1", result.Checks)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runFinalCheckCommands did not return promptly after parent context cancellation")
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("file %s was not created before deadline", path)
}

func TestFinalChecksConfig_TimeoutDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.FinalChecks.Timeout != 600 {
		t.Errorf("default Timeout = %d, want 600", cfg.FinalChecks.Timeout)
	}
}

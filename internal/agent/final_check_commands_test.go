package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRunFinalCheckCommands_NoCommands(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = nil

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if result.NeedsContinue {
		t.Error("expected needsContinue=false when no final checks are configured")
	}
	if result.Feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.Feedback)
	}
}

func TestRunFinalCheckCommands_SuccessfulCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"echo 'test passed'"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if result.NeedsContinue {
		t.Error("expected needsContinue=false for successful command")
	}
	if result.Feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.Feedback)
	}
}

func TestRunFinalCheckCommands_FailedCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
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
}

func TestRunFinalCheckCommands_ChangedFilesEnv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{`test -n "$XELYON_CHANGED_FILES"`}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go", "/src/util.go"})
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
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if !result.NeedsContinue {
		t.Error("expected needsContinue=true when second command fails")
	}
	if !strings.Contains(result.Feedback, "exit code 42") {
		t.Errorf("expected exit code 42 in feedback, got %q", result.Feedback)
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
	result := a.runFinalCheckCommands([]string{"src/main.go"})
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
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if !result.NeedsContinue {
		t.Error("expected needsContinue=true for timed-out command")
	}
	if result.Feedback == "" {
		t.Error("expected non-empty feedback for timed-out command")
	}
}

func TestFinalChecksConfig_TimeoutDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.FinalChecks.Timeout != 600 {
		t.Errorf("default Timeout = %d, want 600", cfg.FinalChecks.Timeout)
	}
}

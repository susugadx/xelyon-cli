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
	if result.needsContinue {
		t.Error("expected needsContinue=false when no final checks are configured")
	}
	if result.feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.feedback)
	}
}

func TestRunFinalCheckCommands_SuccessfulCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"echo 'test passed'"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if result.needsContinue {
		t.Error("expected needsContinue=false for successful command")
	}
	if result.feedback != "" {
		t.Errorf("expected empty feedback, got %q", result.feedback)
	}
}

func TestRunFinalCheckCommands_FailedCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"exit 1"}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if !result.needsContinue {
		t.Error("expected needsContinue=true for failed command")
	}
	if !strings.Contains(result.feedback, "[SYSTEM] Final check failed") {
		t.Errorf("expected system feedback, got %q", result.feedback)
	}
	if !strings.Contains(result.feedback, "exit code 1") {
		t.Errorf("expected exit code in feedback, got %q", result.feedback)
	}
	if result.failureFingerprint == "" {
		t.Error("expected non-empty failure fingerprint")
	}
}

func TestRunFinalCheckCommands_ChangedFilesEnv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{`test -n "$XELYON_CHANGED_FILES"`}
	cfg.FinalChecks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go", "/src/util.go"})
	if result.needsContinue {
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
	if !result.needsContinue {
		t.Error("expected needsContinue=true when second command fails")
	}
	if !strings.Contains(result.feedback, "exit code 42") {
		t.Errorf("expected exit code 42 in feedback, got %q", result.feedback)
	}
}

func TestRunFinalCheckCommands_Timeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"sleep 30"}
	cfg.FinalChecks.Timeout = 1

	a := newCompletionTestAgent(cfg)
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if !result.needsContinue {
		t.Error("expected needsContinue=true for timed-out command")
	}
	if result.feedback == "" {
		t.Error("expected non-empty feedback for timed-out command")
	}
}

func TestFinalChecksConfig_TimeoutDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.FinalChecks.Timeout != 600 {
		t.Errorf("default Timeout = %d, want 600", cfg.FinalChecks.Timeout)
	}
}

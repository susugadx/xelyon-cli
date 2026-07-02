package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_ResumeUsesTUIPath(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	tuiCalled := false
	legacyCalled := false
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		tuiCalled = true
		return nil
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}

	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !tuiCalled {
		t.Fatal("expected --resume to use TUI resume path")
	}
	if legacyCalled {
		t.Fatal("legacy resume path must not be executed by default")
	}
}

func TestRootCommand_ResumePropagatesTUIError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		return fmt.Errorf("resume failed")
	}

	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected --resume error")
	}
	if !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("error = %v, want resume failed", err)
	}
}

func TestRootCommand_NoTUIResumeUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	tuiCalled := false
	legacyCalled := false
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		tuiCalled = true
		return nil
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}

	rootCmd.SetArgs([]string{"--resume", "--no-tui", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !legacyCalled {
		t.Fatal("expected --no-tui --resume to use legacy resume path")
	}
	if tuiCalled {
		t.Fatal("TUI resume path must not be executed when --no-tui is set")
	}
}

func TestResumeCommand_DefaultOpensPicker(t *testing.T) {
	withRootCommandTest(t)

	var pickerCalled bool
	var pickerAll bool
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
		pickerAll = all
	}

	rootCmd.SetArgs([]string{"resume", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !pickerCalled {
		t.Fatal("expected resume to open picker")
	}
	if pickerAll {
		t.Fatal("resume without --all should use cwd-scoped picker")
	}
}

func TestResumeCommand_AllOpensAllSessionPicker(t *testing.T) {
	withRootCommandTest(t)

	var pickerCalled bool
	var pickerAll bool
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
		pickerAll = all
	}

	rootCmd.SetArgs([]string{"resume", "--all", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !pickerCalled || !pickerAll {
		t.Fatalf("pickerCalled=%v pickerAll=%v, want all picker", pickerCalled, pickerAll)
	}
}

func TestResumeCommand_NoTUIUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	var legacyCalled bool
	var pickerCalled bool
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}
	runTUIWithResumePicker = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
		pickerCalled = true
	}

	rootCmd.SetArgs([]string{"resume", "--no-tui", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !legacyCalled {
		t.Fatal("expected resume --no-tui to use legacy resume path")
	}
	if pickerCalled {
		t.Fatal("resume picker must not run when --no-tui is set")
	}
}

func TestResumeCommand_NoTUIRejectsDirectSessionID(t *testing.T) {
	withRootCommandTest(t)

	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		t.Fatal("legacy resume path must not run for direct session ID")
	}

	rootCmd.SetArgs([]string{"resume", "--no-tui", "session-42", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected resume --no-tui direct session ID error")
	}
	if !strings.Contains(err.Error(), "resume session picker and direct session IDs require TUI") {
		t.Fatalf("error = %v, want TUI-required message", err)
	}
}

func TestResumeCommand_NoTUIRejectsAllSessionPicker(t *testing.T) {
	withRootCommandTest(t)

	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		t.Fatal("legacy resume path must not run for --all picker")
	}

	rootCmd.SetArgs([]string{"resume", "--no-tui", "--all", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected resume --no-tui --all error")
	}
	if !strings.Contains(err.Error(), "resume session picker and direct session IDs require TUI") {
		t.Fatalf("error = %v, want TUI-required message", err)
	}
}

func TestResumeCommand_LastUsesResumePath(t *testing.T) {
	withRootCommandTest(t)

	var resumeCalled bool
	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		resumeCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resumeCalled {
		t.Fatal("expected resume --last to use last-session path")
	}
}

func TestResumeCommand_LastPropagatesResumeError(t *testing.T) {
	withRootCommandTest(t)

	runTUIWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
		return fmt.Errorf("resume last failed")
	}

	rootCmd.SetArgs([]string{"resume", "--last", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected resume --last error")
	}
	if !strings.Contains(err.Error(), "resume last failed") {
		t.Fatalf("error = %v, want resume last failed", err)
	}
}

func TestResumeCommand_SessionIDUsesDirectPath(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	sessionID := seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	var gotSessionID string
	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, sessionID string) error {
		gotSessionID = sessionID
		return nil
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotSessionID != sessionID {
		t.Fatalf("sessionID = %q, want %q", gotSessionID, sessionID)
	}
}

func TestResumeCommand_DirectPathPropagatesError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	sessionID := seedOllamaResumeSession(t, "qwen2.5-coder:14b")

	runTUIWithResumeDirect = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool, sessionID string) error {
		return fmt.Errorf("resume failed")
	}

	rootCmd.SetArgs([]string{"resume", sessionID, "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected direct resume error")
	}
	if !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("error = %v, want resume failed", err)
	}
}

func TestResumeCommand_RejectsAllWithSessionID(t *testing.T) {
	withRootCommandTest(t)

	rootCmd.SetArgs([]string{"resume", "--all", "session-42", "--provider", "ollama", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --all with session ID")
	}
	if !strings.Contains(err.Error(), "--all cannot be used with a session ID") {
		t.Fatalf("error = %v, want --all/session-id conflict", err)
	}
}

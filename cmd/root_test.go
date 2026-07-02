package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_PositionalQueryDefaultsToOnce(t *testing.T) {
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		interactiveCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "", nil, 0)
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		interactiveCalled = true
		return nil
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !onceCalled {
		t.Fatal("expected positional query to use one-shot path")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed for positional one-shot query")
	}
}

func TestRootCommand_OnceExecutesSingleTurn(t *testing.T) {
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runLegacyInteractiveWithResume = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		interactiveCalled = true
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool, quiet bool) error {
		interactiveCalled = true
		return nil
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !onceCalled {
		t.Fatal("expected --once path to be executed")
	}
	if interactiveCalled {
		t.Fatal("interactive path must not be executed when --once is set")
	}
}

func TestRootCommand_InteractiveFlagForcesTUIWithPositionalQuery(t *testing.T) {
	withRootCommandTest(t)

	interactiveCalled := false
	onceCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runTUI = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		interactiveCalled = true
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		return nil
	}

	rootCmd.SetArgs([]string{"--interactive", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !interactiveCalled {
		t.Fatal("expected --interactive to use interactive path")
	}
	if onceCalled {
		t.Fatal("one-shot path must not be executed when --interactive is set")
	}
}

func TestRootCommand_NoTUIInteractiveUsesLegacyPath(t *testing.T) {
	withRootCommandTest(t)

	legacyCalled := false
	tuiCalled := false
	runLegacyInteractive = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		legacyCalled = true
	}
	runTUI = func(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
		tuiCalled = true
	}

	rootCmd.SetArgs([]string{"--interactive", "--no-tui", "--provider", "ollama", "--no-update-check"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !legacyCalled {
		t.Fatal("expected --no-tui --interactive to use legacy path")
	}
	if tuiCalled {
		t.Fatal("TUI path must not be executed when --no-tui is set")
	}
}

func TestRootCommand_InteractiveWithOnceReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--interactive", "--once", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --interactive with --once")
	}
	if !strings.Contains(err.Error(), "--interactive cannot be used with --once") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_OnceRequiresQuery(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --once has no query")
	}
}

func TestRootCommand_OnceWithResumeReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--resume", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --resume")
	}
}

func TestRootCommand_ResumeWithPositionalQueryReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--resume", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --resume with positional query")
	}
	if !strings.Contains(err.Error(), "--resume cannot be used with query arguments") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_OnceWithHeadlessReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--once", "--headless", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --headless")
	}
}

func TestRootCommand_QuietWithoutOnceReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--quiet", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --quiet without one-shot execution")
	}
	if !strings.Contains(err.Error(), "--quiet can only be used with one-shot execution") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_InteractiveWithQuietReturnsError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--interactive", "--quiet", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --interactive with --quiet")
	}
	if !strings.Contains(err.Error(), "--quiet can only be used with one-shot execution") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_QuietWithPositionalQueryUsesImplicitOnce(t *testing.T) {
	withRootCommandTest(t)

	called := false
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		called = true
		if !quiet {
			t.Fatal("expected quiet to be passed to one-shot execution")
		}
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--quiet", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !called {
		t.Fatal("expected implicit one-shot execution")
	}
}

func TestRootCommand_OnceMultiWordQuery(t *testing.T) {
	withRootCommandTest(t)

	var gotQuery string
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		gotQuery = query
		return nil
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "fix", "this", "bug"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "fix this bug"
	if gotQuery != want {
		t.Fatalf("query = %q, want %q", gotQuery, want)
	}
}

func TestRootCommand_OnceErrorPropagation(t *testing.T) {
	withRootCommandTest(t)

	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		return fmt.Errorf("something went wrong")
	}

	rootCmd.SetArgs([]string{"--once", "--provider", "ollama", "--no-update-check", "hello"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error to propagate from runOnce")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Fatalf("unexpected error: %v", err)
	}
}

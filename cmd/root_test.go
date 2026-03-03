package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
)

func resetRootFlagsForTest() {
	resume = false
	once = false
	quiet = false
	providerFlag = ""
	modelFlag = ""
	autoApprove = false
	loopThreshold = 0
	apiRetry = 0
	apiRetryDelay = 0
	diffLines = -1
	outputFormat = "text"
	headless = false
	noUpdateCheck = false
	imageFlag = ""
}

func TestRootCommand_OnceExecutesSingleTurn(t *testing.T) {
	resetRootFlagsForTest()

	origRunInteractive := runInteractive
	origRunInteractiveWithResume := runInteractiveWithResume
	origRunHeadless := runHeadless
	origRunOnce := runOnce
	origRunOnceWithImage := runOnceWithImage
	t.Cleanup(func() {
		runInteractive = origRunInteractive
		runInteractiveWithResume = origRunInteractiveWithResume
		runHeadless = origRunHeadless
		runOnce = origRunOnce
		runOnceWithImage = origRunOnceWithImage
		resetRootFlagsForTest()
	})

	interactiveCalled := false
	onceCalled := false
	runInteractive = func(model string, provider api.Provider, autoApprove bool) {
		interactiveCalled = true
	}
	runInteractiveWithResume = func(model string, provider api.Provider, autoApprove bool) {
		interactiveCalled = true
	}
	runHeadless = func(query string, model string, provider api.Provider) *agent.HeadlessResult {
		interactiveCalled = true
		return nil
	}
	runOnceWithImage = func(query string, model string, provider api.Provider, imagePath string, autoApprove bool) {
		interactiveCalled = true
	}
	runOnce = func(query string, model string, provider api.Provider, autoApprove bool, quiet bool) error {
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

func TestRootCommand_OnceRequiresQuery(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --once has no query")
	}
}

func TestRootCommand_OnceWithResumeReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--resume", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --resume")
	}
}

func TestRootCommand_OnceWithHeadlessReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--once", "--headless", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --once with --headless")
	}
}

func TestRootCommand_QuietWithoutOnceReturnsError(t *testing.T) {
	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--quiet", "--no-update-check"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --quiet without --once")
	}
	if !strings.Contains(err.Error(), "--quiet can only be used with --once") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRootCommand_OnceMultiWordQuery(t *testing.T) {
	resetRootFlagsForTest()

	origRunOnce := runOnce
	t.Cleanup(func() {
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	var gotQuery string
	runOnce = func(query string, model string, provider api.Provider, autoApprove bool, quiet bool) error {
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
	resetRootFlagsForTest()

	origRunOnce := runOnce
	t.Cleanup(func() {
		runOnce = origRunOnce
		resetRootFlagsForTest()
	})

	runOnce = func(query string, model string, provider api.Provider, autoApprove bool, quiet bool) error {
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

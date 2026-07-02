package cmd

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_FailOnToolErrorFlagIsIgnoredOutsideHeadless(t *testing.T) {
	withRootCommandTest(t)

	onceCalled := false
	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}
	runOnce = func(query string, model string, provider api.Provider, cfg *config.Config, autoApprove bool, quiet bool) error {
		onceCalled = true
		if query != "hello" {
			t.Fatalf("query = %q, want hello", query)
		}
		return nil
	}

	rootCmd.SetArgs([]string{"--fail-on-tool-error", "--provider", "ollama", "--no-update-check", "hello"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !onceCalled {
		t.Fatal("one-shot runner was not called")
	}
	if headlessCalled {
		t.Fatal("headless runner must not be called outside headless mode")
	}
}

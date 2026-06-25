package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_HeadlessExitCodePolicyCIOnSuccess(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if parsed.FailureReason != "" {
		t.Fatalf("failure_reason = %q, want empty success reason", parsed.FailureReason)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyCI {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyCI)
	}
	if parsed.RecommendedExitCode != 0 {
		t.Fatalf("recommended_exit_code = %d, want 0", parsed.RecommendedExitCode)
	}
}

func TestRootCommand_HeadlessPromptInputErrorUsesCIExitPolicy(t *testing.T) {
	withRootCommandTest(t)

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check"}, "")
	if err == nil {
		t.Fatal("expected headless execution error")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyCI {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyCI)
	}
	if parsed.RecommendedExitCode != 2 {
		t.Fatalf("recommended_exit_code = %d, want 2", parsed.RecommendedExitCode)
	}
	requireCommandExitCode(t, err, 2)
}

func TestRootCommand_HeadlessProviderSetupRequiredUsesCIExitPolicy(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
		headlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "openai", "--no-update-check", "hello"}, "")
	if err == nil {
		t.Fatal("expected headless setup error")
	}
	if headlessCalled {
		t.Fatal("headless runner must not be called without provider credential")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless setup JSON:\n%s", stderr)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeProviderSetupRequired {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeProviderSetupRequired)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonProviderSetupRequired {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonProviderSetupRequired)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyCI {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyCI)
	}
	if parsed.RecommendedExitCode != 3 {
		t.Fatalf("recommended_exit_code = %d, want 3", parsed.RecommendedExitCode)
	}
	requireCommandExitCode(t, err, 3)
}

func TestRootCommand_HeadlessExitCodePolicyCIClassifiesRuntimeErrors(t *testing.T) {
	tests := []struct {
		name      string
		result    func(provider api.Provider, model string) *agent.HeadlessResult
		errorType string
		reason    agent.HeadlessFailureReason
		code      int
	}{
		{
			name: "api error",
			result: func(provider api.Provider, model string) *agent.HeadlessResult {
				return agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeAPI, "api failed", 0)
			},
			errorType: agent.HeadlessErrorTypeAPI,
			reason:    agent.HeadlessFailureReasonAPIError,
			code:      6,
		},
		{
			name: "cancelled",
			result: func(provider api.Provider, model string) *agent.HeadlessResult {
				return agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeCancelled, "context canceled", 0)
			},
			errorType: agent.HeadlessErrorTypeCancelled,
			reason:    agent.HeadlessFailureReasonCancelled,
			code:      7,
		},
		{
			name: "tool loop limit",
			result: func(provider api.Provider, model string) *agent.HeadlessResult {
				return agent.NewToolLoopLimitResult(provider.Name(), model, 10, nil, 0)
			},
			errorType: agent.HeadlessErrorTypeToolLoopLimit,
			reason:    agent.HeadlessFailureReasonToolLoopLimit,
			code:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			t.Setenv("HOME", t.TempDir())

			runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config) *agent.HeadlessResult {
				return tt.result(provider, model)
			}

			parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}, "")
			if err == nil {
				t.Fatal("expected headless execution error")
			}
			if strings.Contains(stderr, "Usage:") {
				t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
			}
			if parsed.Error == nil || parsed.Error.Type != tt.errorType {
				t.Fatalf("error = %+v, want %s", parsed.Error, tt.errorType)
			}
			if parsed.FailureReason != tt.reason {
				t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, tt.reason)
			}
			if parsed.ExitPolicy != agent.HeadlessExitPolicyCI {
				t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyCI)
			}
			if parsed.RecommendedExitCode != tt.code {
				t.Fatalf("recommended_exit_code = %d, want %d", parsed.RecommendedExitCode, tt.code)
			}
			requireCommandExitCode(t, err, tt.code)
		})
	}
}

func TestRootCommand_InvalidExitCodePolicyReturnsCommandError(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--exit-code-policy", "strict", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid exit code policy")
	}
	if !strings.Contains(err.Error(), "invalid --exit-code-policy") {
		t.Fatalf("unexpected error message: %v", err)
	}
	var exitErr exitCodeCarrier
	if errors.As(err, &exitErr) {
		t.Fatalf("invalid exit-code-policy error carries exit code %d, want normal command error", exitErr.ExitCode())
	}
}

func requireCommandExitCode(t *testing.T, err error, want int) {
	t.Helper()
	var exitErr exitCodeCarrier
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want exit code carrier", err)
	}
	if exitErr.ExitCode() != want {
		t.Fatalf("ExitCode() = %d, want %d", exitErr.ExitCode(), want)
	}
}

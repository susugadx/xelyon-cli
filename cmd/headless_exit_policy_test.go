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

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if options.FailOnToolError {
			t.Fatal("FailOnToolError = true, want default false")
		}
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

func TestRootCommand_HeadlessDefaultsFailOnToolErrorFalse(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if options.FailOnToolError {
			t.Fatal("FailOnToolError = true, want default false")
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", []agent.ToolCallResult{{
			Tool:    "str_replace",
			Args:    map[string]string{"path": "target.txt"},
			Output:  "Error: old_str not found",
			Success: false,
		}}, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if parsed.Status != agent.HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success", parsed.Status)
	}
	if parsed.FailureReason != "" {
		t.Fatalf("failure_reason = %q, want empty", parsed.FailureReason)
	}
	if parsed.RecommendedExitCode != 0 {
		t.Fatalf("recommended_exit_code = %d, want 0", parsed.RecommendedExitCode)
	}
	if len(parsed.ToolCalls) != 1 || parsed.ToolCalls[0].Success {
		t.Fatalf("tool_calls = %+v, want preserved failed tool call", parsed.ToolCalls)
	}
}

func TestRootCommand_FailOnToolErrorUsesCIExitPolicy(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	called := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		called = true
		if !options.FailOnToolError {
			t.Fatal("FailOnToolError = false, want true")
		}
		result := agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeToolError, "one or more tool calls failed", 0)
		result.Response = "final response after tool failure"
		result.ToolCalls = []agent.ToolCallResult{{
			Tool:    "str_replace",
			Args:    map[string]string{"path": "target.txt"},
			Output:  "Error: old_str not found",
			Success: false,
		}}
		return result
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--fail-on-tool-error", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err == nil {
		t.Fatal("expected headless execution error")
	}
	if !called {
		t.Fatal("headless runner was not called")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want error", parsed.Status)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeToolError {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeToolError)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonToolError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonToolError)
	}
	if parsed.RecommendedExitCode != 4 {
		t.Fatalf("recommended_exit_code = %d, want 4", parsed.RecommendedExitCode)
	}
	if parsed.Response != "final response after tool failure" {
		t.Fatalf("response = %q, want preserved final response", parsed.Response)
	}
	if len(parsed.ToolCalls) != 1 || parsed.ToolCalls[0].Success {
		t.Fatalf("tool_calls = %+v, want preserved failed tool call", parsed.ToolCalls)
	}
	requireCommandExitCode(t, err, 4)
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

func TestRootCommand_HeadlessMissingProviderCredentialPrintsJSONSetupError(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "openai", "--no-update-check", "hello"}, "")
	if err == nil {
		t.Fatal("expected headless execution error")
	}
	requireCommandExitCode(t, err, 1)
	if headlessCalled {
		t.Fatal("headless runner must not be called without provider credential")
	}
	if !rootCmd.SilenceUsage {
		t.Fatal("rootCmd.SilenceUsage = false, want true after printing headless setup JSON")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless setup JSON:\n%s", stderr)
	}
	if parsed.SchemaVersion != agent.HeadlessSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", parsed.SchemaVersion, agent.HeadlessSchemaVersion)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceArgs, "", len([]byte("hello")))
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeProviderSetupRequired {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeProviderSetupRequired)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonProviderSetupRequired {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonProviderSetupRequired)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyLegacy {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyLegacy)
	}
	if parsed.RecommendedExitCode != 1 {
		t.Fatalf("recommended_exit_code = %d, want 1", parsed.RecommendedExitCode)
	}
	for _, fragment := range []string{"OPENAI_API_KEY", "xelyon setup"} {
		if !strings.Contains(parsed.Error.Message, fragment) {
			t.Fatalf("setup JSON error missing %q:\n%s", fragment, parsed.Error.Message)
		}
	}
}

func TestRootCommand_HeadlessProviderSetupRequiredUsesCIExitPolicy(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
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

func TestRootCommand_HeadlessUnknownProviderReturnsConfigJSONWithoutSetupJSON(t *testing.T) {
	withRootCommandTest(t)

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "not-a-provider", "--no-update-check", "hello"}, "")
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	requireCommandExitCode(t, err, 3)
	if headlessCalled {
		t.Fatal("headless runner must not be called with unknown provider")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if output == "" {
		t.Fatal("stdout is empty, want headless JSON")
	}
	if parsed.Provider != "not-a-provider" {
		t.Fatalf("provider = %q, want not-a-provider", parsed.Provider)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonConfigError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonConfigError)
	}
	if parsed.RecommendedExitCode != 3 {
		t.Fatalf("recommended_exit_code = %d, want 3", parsed.RecommendedExitCode)
	}
	if strings.Contains(output, agent.HeadlessErrorTypeProviderSetupRequired) {
		t.Fatalf("stdout must not contain provider setup JSON: %q", output)
	}
	if !strings.Contains(parsed.Error.Message, "unknown provider") {
		t.Fatalf("error message = %q, want unknown provider", parsed.Error.Message)
	}
}

func TestRootCommand_HeadlessModelValidationErrorUsesConfigJSON(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GEMINI_API_KEY", "test-key")

	headlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		headlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--provider", "gemini", "--model", "gemini-2.0-flash-lite", "--no-update-check", "hello"}, "")
	if err == nil {
		t.Fatal("expected headless model validation error")
	}
	if headlessCalled {
		t.Fatal("headless runner must not be called after model validation error")
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless config JSON:\n%s", stderr)
	}
	if parsed.Provider != "Gemini" {
		t.Fatalf("provider = %q, want Gemini", parsed.Provider)
	}
	if parsed.Model != "gemini-2.0-flash-lite" {
		t.Fatalf("model = %q, want gemini-2.0-flash-lite", parsed.Model)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonConfigError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonConfigError)
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
		{
			name: "final check failed",
			result: func(provider api.Provider, model string) *agent.HeadlessResult {
				return agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeFinalCheckFailed, "final checks failed", 0)
			},
			errorType: agent.HeadlessErrorTypeFinalCheckFailed,
			reason:    agent.HeadlessFailureReasonFinalCheckFailed,
			code:      5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			t.Setenv("HOME", t.TempDir())

			runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
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

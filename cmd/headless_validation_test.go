package cmd

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
)

func TestRootCommand_HeadlessIntentValidationErrorsReturnJSON(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "headless once",
			args:      []string{"--headless", "--exit-code-policy", "ci", "--once", "--provider", "ollama", "--no-update-check", "hello"},
			wantError: "--once cannot be used with --headless or --output-format json",
		},
		{
			name:      "json once",
			args:      []string{"--output-format", "json", "--exit-code-policy", "ci", "--once", "--provider", "ollama", "--no-update-check", "hello"},
			wantError: "--once cannot be used with --headless or --output-format json",
		},
		{
			name:      "headless invalid output format",
			args:      []string{"--headless", "--output-format", "yaml", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"},
			wantError: "invalid --output-format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			t.Setenv("HOME", t.TempDir())

			parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, tt.args, "")
			if err == nil {
				t.Fatal("expected headless validation error")
			}
			requireCommandExitCode(t, err, 2)
			if output == "" {
				t.Fatal("stdout is empty, want headless JSON")
			}
			if strings.Contains(stderr, "Usage:") {
				t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
			}
			if parsed.Status != agent.HeadlessStatusError {
				t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
			}
			if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
				t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
			}
			if !strings.Contains(parsed.Error.Message, tt.wantError) {
				t.Fatalf("error message = %q, want %q", parsed.Error.Message, tt.wantError)
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
			requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceArgs, "", len([]byte("hello")))
		})
	}
}

func TestRootCommand_HeadlessIntentValidationImageErrorIncludesMetadata(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	imagePath := "/tmp/missing-is-not-read.png"
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--exit-code-policy", "ci", "--resume", "--image", imagePath, "--provider", "ollama", "--no-update-check"}, "")
	if err == nil {
		t.Fatal("expected headless validation error")
	}
	requireCommandExitCode(t, err, 2)
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.Error == nil || !strings.Contains(parsed.Error.Message, "--resume cannot be used with --image") {
		t.Fatalf("error = %+v, want --resume image validation error", parsed.Error)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceArgs, "", 0)
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, false)
}

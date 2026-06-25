package cmd

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestExecute_HelperProcess(t *testing.T) {
	mode := os.Getenv("GO_WANT_XELYON_ROOT_EXECUTE_HELPER")
	if mode == "" {
		return
	}

	resetRootFlagsForTest()
	var args []string
	switch mode {
	case "unknown_flag", "1":
		args = []string{"--unknown-flag"}
	case "unknown_flag_ci":
		args = []string{"--exit-code-policy", "ci", "--unknown-flag"}
	case "unknown_flag_then_ci":
		args = []string{"--unknown-flag", "--exit-code-policy", "ci"}
	case "unknown_flag_then_ci_equals":
		args = []string{"--unknown-flag", "--exit-code-policy=ci"}
	case "unknown_shorthand_flag_ci":
		args = []string{"--exit-code-policy", "ci", "-z"}
	case "missing_flag_argument_ci":
		args = []string{"--exit-code-policy", "ci", "--provider"}
	case "headless_usage_ci":
		args = []string{"--headless", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check"}
	case "headless_tool_error_ci":
		runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
			if !options.FailOnToolError {
				t.Fatal("FailOnToolError = false, want true")
			}
			result := agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeToolError, "one or more tool calls failed", 0)
			result.ToolCalls = []agent.ToolCallResult{{
				Tool:    "str_replace",
				Args:    map[string]string{"path": "target.txt"},
				Output:  "Error: old_str not found",
				Success: false,
			}}
			return result
		}
		args = []string{"--headless", "--fail-on-tool-error", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}
	case "root_usage_ci":
		args = []string{"--exit-code-policy", "ci", "--output-format", "yaml", "--no-update-check", "hello"}
	case "invalid_exit_policy":
		args = []string{"--exit-code-policy", "strict", "--no-update-check", "hello"}
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
	os.Args = append([]string{os.Args[0]}, args...)
	Execute()
}

func TestExecute_ExitsOnError(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=unknown_flag")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(string(output), "unknown flag") {
		t.Fatalf("combined output = %q, want cobra error message", string(output))
	}
}

func TestExecute_ExitsWithCIUsageCodeForCobraUsageErrors(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	tests := []struct {
		name         string
		helperMode   string
		wantFragment string
	}{
		{name: "unknown long flag", helperMode: "unknown_flag_ci", wantFragment: "unknown flag"},
		{name: "unknown long flag before policy", helperMode: "unknown_flag_then_ci", wantFragment: "unknown flag"},
		{name: "unknown long flag before equals policy", helperMode: "unknown_flag_then_ci_equals", wantFragment: "unknown flag"},
		{name: "unknown shorthand flag", helperMode: "unknown_shorthand_flag_ci", wantFragment: "unknown shorthand flag"},
		{name: "missing flag argument", helperMode: "missing_flag_argument_ci", wantFragment: "flag needs an argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
			cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER="+tt.helperMode)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("expected Execute() helper to exit with non-zero status")
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("error = %T, want *exec.ExitError", err)
			}
			if exitErr.ExitCode() != 2 {
				t.Fatalf("exit code = %d, want 2\noutput=%s", exitErr.ExitCode(), string(output))
			}
			if !strings.Contains(string(output), tt.wantFragment) {
				t.Fatalf("combined output = %q, want %q", string(output), tt.wantFragment)
			}
		})
	}
}

func TestExecute_ExitsWithHeadlessRecommendedCode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=headless_usage_ci")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("exit code = %d, want 2\noutput=%s", exitErr.ExitCode(), string(output))
	}
	if !strings.Contains(string(output), `"failure_reason": "usage_error"`) {
		t.Fatalf("combined output = %q, want usage_error JSON", string(output))
	}
	if !strings.Contains(string(output), `"recommended_exit_code": 2`) {
		t.Fatalf("combined output = %q, want recommended_exit_code 2", string(output))
	}
}

func TestExecute_ExitsWithHeadlessToolErrorRecommendedCode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=headless_tool_error_ci")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 4 {
		t.Fatalf("exit code = %d, want 4\noutput=%s", exitErr.ExitCode(), string(output))
	}
	if !strings.Contains(string(output), `"failure_reason": "tool_error"`) {
		t.Fatalf("combined output = %q, want tool_error JSON", string(output))
	}
	if !strings.Contains(string(output), `"recommended_exit_code": 4`) {
		t.Fatalf("combined output = %q, want recommended_exit_code 4", string(output))
	}
}

func TestExecute_ExitsWithCIUsageCodeForRootErrors(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=root_usage_ci")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Fatalf("exit code = %d, want 2\noutput=%s", exitErr.ExitCode(), string(output))
	}
	if !strings.Contains(string(output), "invalid --output-format") {
		t.Fatalf("combined output = %q, want output-format error", string(output))
	}
}

func TestExecute_InvalidExitPolicyKeepsLegacyErrorCode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=invalid_exit_policy")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1\noutput=%s", exitErr.ExitCode(), string(output))
	}
	if !strings.Contains(string(output), "invalid --exit-code-policy") {
		t.Fatalf("combined output = %q, want exit policy error", string(output))
	}
}

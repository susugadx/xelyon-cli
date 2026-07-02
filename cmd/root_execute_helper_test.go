package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type rootExecuteHelperRun struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r rootExecuteHelperRun) combinedOutput() string {
	return r.stdout + r.stderr
}

func runRootExecuteHelper(t *testing.T, mode string) rootExecuteHelperRun {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER="+mode)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		return rootExecuteHelperRun{stdout: stdout.String(), stderr: stderr.String()}
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	return rootExecuteHelperRun{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitErr.ExitCode(),
	}
}

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
	case "headless_unknown_flag_ci":
		args = []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "prompt"}
	case "headless_unknown_flag_ci_policy_after":
		args = []string{"--headless", "--bad-flag", "--exit-code-policy", "ci", "prompt"}
	case "headless_unknown_flag_ci_headless_false_after":
		args = []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--headless=false", "prompt"}
	case "headless_unknown_flag_ci_before_doctor":
		args = []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "doctor"}
	case "headless_unknown_shorthand_cluster_ci":
		args = []string{"--headless", "--exit-code-policy", "ci", "-qz", "prompt"}
	case "json_unknown_flag_ci_text_after":
		args = []string{"--output-format", "json", "--exit-code-policy", "ci", "--bad-flag", "--output-format", "text", "prompt"}
	case "headless_unknown_flag_ci_prompt_file_after":
		args = []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--prompt-file", "prompt.md"}
	case "headless_unknown_flag_ci_image_after":
		args = []string{"--headless", "--exit-code-policy", "ci", "--bad-flag", "--provider", "openai", "--image", "screen.png"}
	case "headless_unknown_flag_ci_image_attached_shorthand":
		args = []string{"--headless", "--exit-code-policy", "ci", "-popenai", "-iscreen.png", "--bad-flag", "prompt"}
	case "headless_unknown_flag_ci_image_equals_shorthand":
		args = []string{"--headless", "--exit-code-policy", "ci", "-p=groq", "-i=screen.png", "--bad-flag", "prompt"}
	case "headless_unknown_flag_ci_model_cluster":
		args = []string{"--headless", "--exit-code-policy", "ci", "-qm", "gpt", "--bad-flag", "prompt"}
	case "headless_unknown_flag_ci_image_cluster":
		args = []string{"--headless", "--exit-code-policy", "ci", "--provider", "openai", "-qiscreen.png", "--bad-flag", "prompt"}
	case "json_unknown_flag_ci_uppercase":
		args = []string{"--output-format", "JSON", "--exit-code-policy", "ci", "--bad-flag", "prompt"}
	case "json_unknown_flag_ci_whitespace_policy_after":
		args = []string{"--output-format", " json ", "--bad-flag", "--exit-code-policy", "ci", "prompt"}
	case "headless_false_unknown_flag":
		args = []string{"--headless=false", "--bad-flag", "prompt"}
	case "unknown_flag_then_headless":
		args = []string{"--bad-flag", "--headless", "prompt"}
	case "quiet_cluster_then_headless":
		args = []string{"-qz", "--headless", "prompt"}
	case "auto_approve_cluster_then_headless":
		args = []string{"-yz", "--headless", "prompt"}
	case "headless_before_doctor":
		args = []string{"--headless", "doctor"}
	case "json_before_doctor":
		args = []string{"--output-format", "json", "doctor"}
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
	case "headless_read_only_violation_ci":
		runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
			if !options.FailOnToolError {
				t.Fatal("FailOnToolError = false, want true")
			}
			if !options.ReadOnly {
				t.Fatal("ReadOnly = false, want true")
			}
			result := agent.NewErrorResult(provider.Name(), model, agent.HeadlessErrorTypeReadOnlyViolation, "one or more tool calls were denied by read-only mode", 0)
			result.ToolCalls = []agent.ToolCallResult{{
				Tool:    "write_file",
				Args:    map[string]string{"path": "target.txt"},
				Output:  "Error: read-only mode denied write-capable tool: write_file",
				Success: false,
			}}
			return result
		}
		args = []string{"--headless", "--read-only", "--fail-on-tool-error", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"}
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

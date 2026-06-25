package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRootCommand_HeadlessPromptFilePassesContentAndInputMetadata(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	body := "line one\nline two\n"
	path := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotQuery string
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		gotQuery = query
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, output, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "ollama", "--no-update-check", "--prompt-file", path}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if gotQuery != body {
		t.Fatalf("query = %q, want file body %q", gotQuery, body)
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourcePromptFile, path, len([]byte(body)))
	if strings.Contains(output, body) {
		t.Fatalf("stdout JSON leaked prompt body: %q", output)
	}
}

func TestRootCommand_HeadlessPromptFileDashReadsStdin(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	body := "stdin prompt\n"
	var gotQuery string
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		gotQuery = query
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "ollama", "--no-update-check", "--prompt-file", "-"}, body)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if gotQuery != body {
		t.Fatalf("query = %q, want stdin body %q", gotQuery, body)
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceStdin, "", len([]byte(body)))
}

func TestRootCommand_HeadlessBareDashReadsStdin(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	body := "bare dash prompt\n"
	var gotQuery string
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		gotQuery = query
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "ollama", "--no-update-check", "-"}, body)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if gotQuery != body {
		t.Fatalf("query = %q, want stdin body %q", gotQuery, body)
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourceStdin, "", len([]byte(body)))
}

func TestRootCommand_HeadlessPromptFileRejectsPositionalQuery(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	path := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(path, []byte("prompt"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runHeadlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		runHeadlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "ollama", "--no-update-check", "--prompt-file", path, "hello"}, "")
	requireHeadlessConfigError(t, parsed, stderr, err)
	if runHeadlessCalled {
		t.Fatal("headless runner must not be called after prompt input validation error")
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourcePromptFile, path, 0)
}

func TestRootCommand_HeadlessPromptInputValidationErrorsReturnJSON(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		setup      func(t *testing.T) []string
		wantSource agent.HeadlessInputSource
		wantPath   string
	}{
		{
			name: "missing file",
			setup: func(t *testing.T) []string {
				return []string{"--prompt-file", filepath.Join(t.TempDir(), "missing.md")}
			},
			wantSource: agent.HeadlessInputSourcePromptFile,
		},
		{
			name: "directory",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				return []string{"--prompt-file", dir}
			},
			wantSource: agent.HeadlessInputSourcePromptFile,
		},
		{
			name:       "empty stdin",
			args:       []string{"-"},
			stdin:      " \n\t",
			wantSource: agent.HeadlessInputSourceStdin,
		},
		{
			name:       "oversized stdin",
			args:       []string{"-"},
			stdin:      strings.Repeat("x", headlessPromptInputMaxBytes+1),
			wantSource: agent.HeadlessInputSourceStdin,
		},
		{
			name: "oversized file",
			setup: func(t *testing.T) []string {
				path := filepath.Join(t.TempDir(), "large.md")
				if err := os.WriteFile(path, []byte(strings.Repeat("x", headlessPromptInputMaxBytes+1)), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				return []string{"--prompt-file", path}
			},
			wantSource: agent.HeadlessInputSourcePromptFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			t.Setenv("HOME", t.TempDir())

			runHeadlessCalled := false
			runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
				runHeadlessCalled = true
				return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
			}

			args := append([]string{"--headless", "--provider", "ollama", "--no-update-check"}, tt.args...)
			if tt.setup != nil {
				extra := tt.setup(t)
				args = append(args, extra...)
				if len(extra) == 2 && extra[0] == "--prompt-file" {
					tt.wantPath = extra[1]
				}
			}
			parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, args, tt.stdin)
			requireHeadlessConfigError(t, parsed, stderr, err)
			if runHeadlessCalled {
				t.Fatal("headless runner must not be called after prompt input validation error")
			}
			if parsed.Input == nil {
				t.Fatal("input = nil, want validation metadata")
			}
			if parsed.Input.Source != tt.wantSource {
				t.Fatalf("input.source = %q, want %q", parsed.Input.Source, tt.wantSource)
			}
			if tt.wantPath != "" && parsed.Input.PromptFile != tt.wantPath {
				t.Fatalf("input.prompt_file = %q, want %q", parsed.Input.PromptFile, tt.wantPath)
			}
		})
	}
}

func TestRootCommand_PromptFileRequiresHeadlessJSONMode(t *testing.T) {
	withRootCommandTest(t)

	rootCmd.SetArgs([]string{"--prompt-file", "prompt.md", "--no-update-check"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected --prompt-file mode error")
	}
	if !strings.Contains(err.Error(), "--prompt-file can only be used") {
		t.Fatalf("error = %v, want --prompt-file mode error", err)
	}
}

func executeRootCommandForHeadlessJSONTest(t *testing.T, args []string, stdin string) (agent.HeadlessResult, string, string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	os.Stdout = w

	var stderr bytes.Buffer
	rootCmd.SetOut(&stderr)
	rootCmd.SetErr(&stderr)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)

	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := strings.TrimSpace(buf.String())

	var parsed agent.HeadlessResult
	if output != "" {
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("stdout is not headless JSON: %v\noutput=%q", err, output)
		}
	}
	return parsed, output, stderr.String(), execErr
}

func requireHeadlessInput(t *testing.T, input *agent.HeadlessInput, source agent.HeadlessInputSource, path string, bytes int) {
	t.Helper()
	if input == nil {
		t.Fatal("input = nil")
	}
	if input.Source != source {
		t.Fatalf("input.source = %q, want %q", input.Source, source)
	}
	if input.PromptFile != path {
		t.Fatalf("input.prompt_file = %q, want %q", input.PromptFile, path)
	}
	if input.Bytes != bytes {
		t.Fatalf("input.bytes = %d, want %d", input.Bytes, bytes)
	}
}

func requireHeadlessConfigError(t *testing.T, parsed agent.HeadlessResult, stderr string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected headless execution error")
	}
	if !strings.Contains(err.Error(), "headless execution failed") {
		t.Fatalf("error = %v, want headless execution failed", err)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr contains Cobra usage after headless JSON error:\n%s", stderr)
	}
	if parsed.SchemaVersion != agent.HeadlessSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", parsed.SchemaVersion, agent.HeadlessSchemaVersion)
	}
	if parsed.Status != agent.HeadlessStatusError {
		t.Fatalf("status = %q, want %q", parsed.Status, agent.HeadlessStatusError)
	}
	if parsed.Error == nil || parsed.Error.Type != agent.HeadlessErrorTypeConfig {
		t.Fatalf("error = %+v, want %s", parsed.Error, agent.HeadlessErrorTypeConfig)
	}
	if parsed.FailureReason != agent.HeadlessFailureReasonUsageError {
		t.Fatalf("failure_reason = %q, want %q", parsed.FailureReason, agent.HeadlessFailureReasonUsageError)
	}
	if parsed.ExitPolicy != agent.HeadlessExitPolicyLegacy {
		t.Fatalf("exit_policy = %q, want %q", parsed.ExitPolicy, agent.HeadlessExitPolicyLegacy)
	}
	if parsed.RecommendedExitCode != 1 {
		t.Fatalf("recommended_exit_code = %d, want 1", parsed.RecommendedExitCode)
	}
}

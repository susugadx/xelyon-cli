package cmd

import (
	"context"
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

func TestRootCommand_HeadlessPromptInputValidationWithImageIncludesMetadata(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadlessCalled := false
	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		runHeadlessCalled = true
		return agent.NewSuccessResult(provider.Name(), model, "unexpected", nil, 0)
	}

	imagePath := filepath.Join(t.TempDir(), "missing-is-not-read.png")
	promptPath := filepath.Join(t.TempDir(), "missing.md")
	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "openai", "--no-update-check", "--image", imagePath, "--prompt-file", promptPath}, "")
	requireHeadlessConfigError(t, parsed, stderr, err)
	if runHeadlessCalled {
		t.Fatal("headless runner must not be called after prompt input validation error")
	}
	requireHeadlessInput(t, parsed.Input, agent.HeadlessInputSourcePromptFile, promptPath, 0)
	requireHeadlessInputImage(t, parsed.Input, imagePath, "", 0, true)
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

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

func TestRootCommand_HeadlessReadOnlyDoesNotBootstrapMissingConfig(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{name: "read_only", flag: "--read-only"},
		{name: "dry_run", flag: "--dry-run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			home := t.TempDir()
			t.Setenv("HOME", home)

			runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
				if !options.ReadOnly {
					t.Fatal("ReadOnly = false, want true")
				}
				if cfg == nil {
					t.Fatal("cfg = nil, want default config")
				}
				return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
			}

			parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", tt.flag, "--provider", "ollama", "--no-update-check", "hello"}, "")
			if err != nil {
				t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
			}
			if parsed.Status != agent.HeadlessStatusSuccess {
				t.Fatalf("status = %q, want success", parsed.Status)
			}
			assertNoBootstrapConfigFiles(t, home)
		})
	}
}

func TestRootCommand_HeadlessWithoutReadOnlyKeepsConfigBootstrap(t *testing.T) {
	withRootCommandTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if options.ReadOnly {
			t.Fatal("ReadOnly = true, want false")
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if parsed.Status != agent.HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success", parsed.Status)
	}
	assertBootstrapConfigFilesExist(t, home)
}

func TestRootCommand_HeadlessReadOnlyFlagPassesRunOption(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if !options.ReadOnly {
			t.Fatal("ReadOnly = false, want true")
		}
		if options.FailOnToolError {
			t.Fatal("FailOnToolError = true, want default false")
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--headless", "--read-only", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if parsed.Status != agent.HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success", parsed.Status)
	}
}

func TestRootCommand_HeadlessDryRunFlagIsReadOnlyAlias(t *testing.T) {
	withRootCommandTest(t)
	t.Setenv("HOME", t.TempDir())

	runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
		if !options.ReadOnly {
			t.Fatal("ReadOnly = false, want true for --dry-run")
		}
		return agent.NewSuccessResult(provider.Name(), model, "ok", nil, 0)
	}

	parsed, _, stderr, err := executeRootCommandForHeadlessJSONTest(t, []string{"--output-format", "json", "--dry-run", "--provider", "ollama", "--no-update-check", "hello"}, "")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s", err, stderr)
	}
	if parsed.Status != agent.HeadlessStatusSuccess {
		t.Fatalf("status = %q, want success", parsed.Status)
	}
}

func TestRootCommand_ReadOnlyDryRunRequireHeadlessJSONMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "read only once", args: []string{"--read-only", "--provider", "ollama", "--no-update-check", "hello"}},
		{name: "dry run once", args: []string{"--dry-run", "--provider", "ollama", "--no-update-check", "hello"}},
		{name: "read only interactive", args: []string{"--interactive", "--read-only", "--provider", "ollama", "--no-update-check", "hello"}},
		{name: "dry run resume", args: []string{"--resume", "--dry-run", "--provider", "ollama", "--no-update-check"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withRootCommandTest(t)
			runHeadless = func(ctx context.Context, query string, model string, provider api.Provider, cfg *config.Config, options agent.HeadlessRunOptions) *agent.HeadlessResult {
				t.Fatal("headless runner must not be called after read-only mode validation error")
				return nil
			}
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected read-only/dry-run mode error")
			}
			if !strings.Contains(err.Error(), "--read-only and --dry-run can only be used") {
				t.Fatalf("error = %v, want read-only/dry-run mode error", err)
			}
		})
	}
}

func TestRootCommand_ReadOnlyUsageErrorUsesCIExitCode(t *testing.T) {
	withRootCommandTest(t)
	rootCmd.SetArgs([]string{"--read-only", "--exit-code-policy", "ci", "--provider", "ollama", "--no-update-check", "hello"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected read-only mode error")
	}
	if !strings.Contains(err.Error(), "--read-only and --dry-run can only be used") {
		t.Fatalf("error = %v, want read-only/dry-run mode error", err)
	}
	requireCommandExitCode(t, err, 2)
}

func assertNoBootstrapConfigFiles(t *testing.T, home string) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(home, ".xelyon", "config.yaml"),
		filepath.Join(home, ".xelyon", "AGENTS.md"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("bootstrap file %s stat error = %v, want absent", path, err)
		}
	}
}

func assertBootstrapConfigFilesExist(t *testing.T, home string) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(home, ".xelyon", "config.yaml"),
		filepath.Join(home, ".xelyon", "AGENTS.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("bootstrap file %s stat error = %v, want present", path, err)
		}
	}
}

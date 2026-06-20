package probe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestProbeRunner_HostReadOnlyPassed(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != domain.ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, domain.ReviewProbePassed)
	}
}

func TestProbeRunner_HostReadOnlyPassed_GitGlobalOption(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass-git-global-option",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"--no-optional-locks", "status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != domain.ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, domain.ReviewProbePassed)
	}
}

func TestHostReadOnlyExecutor_ChildProcessUsesHardenedEnvAndCleansUp(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createProbeTestScriptCommand(t, safeBin, "git", strings.Join([]string{
		`printf 'HAS_SECRET_TOKEN=%s\n' "${SECRET_TOKEN+x}"`,
		`printf 'HAS_GIT_EXTERNAL_DIFF=%s\n' "${GIT_EXTERNAL_DIFF+x}"`,
		`printf 'HAS_GIT_DIFF_OPTS=%s\n' "${GIT_DIFF_OPTS+x}"`,
		`printf 'HAS_GIT_PAGER=%s\n' "${GIT_PAGER+x}"`,
		`printf 'HAS_PAGER=%s\n' "${PAGER+x}"`,
		`printf 'HAS_RUSTC_WRAPPER=%s\n' "${RUSTC_WRAPPER+x}"`,
		`printf 'HOME=%s\n' "$HOME"`,
		`printf 'TMPDIR=%s\n' "$TMPDIR"`,
		`printf 'GOENV=%s\n' "$GOENV"`,
		`printf 'GIT_CONFIG_GLOBAL=%s\n' "$GIT_CONFIG_GLOBAL"`,
		`printf 'GIT_CONFIG_SYSTEM=%s\n' "$GIT_CONFIG_SYSTEM"`,
		`printf 'NPM_CONFIG_PREFIX=%s\n' "$NPM_CONFIG_PREFIX"`,
		`printf 'CARGO_HOME=%s\n' "$CARGO_HOME"`,
	}, "\n"))

	runtimeRoot := filepath.Join(t.TempDir(), "xelyon-review-host-readonly-env-child")
	executor := newHostReadOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + safeBin,
		"SECRET_TOKEN=secret",
		"GIT_EXTERNAL_DIFF=/tmp/external-diff",
		"GIT_DIFF_OPTS=--unified=0",
		"GIT_PAGER=/tmp/pager",
		"PAGER=/tmp/pager",
		"NPM_CONFIG_PREFIX=/tmp/npm-prefix",
		"npm_config_userconfig=/tmp/npmrc",
		"CARGO_HOME=/tmp/cargo-home",
		"RUSTC_WRAPPER=/tmp/rustc-wrapper",
		"GOENV=/tmp/goenv",
	}
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
			return "", err
		}
		return runtimeRoot, nil
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "host-readonly-env-child",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 4 * 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})

	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbePassed, result.Error, firstCommandOutput(result))
	}
	envMap := parseProbeKeyValueOutput(t, result.CommandResults[0].Output)
	for _, key := range []string{
		"HAS_SECRET_TOKEN",
		"HAS_GIT_EXTERNAL_DIFF",
		"HAS_GIT_DIFF_OPTS",
		"HAS_GIT_PAGER",
		"HAS_PAGER",
		"HAS_RUSTC_WRAPPER",
	} {
		if envMap[key] != "" {
			t.Fatalf("%s = %q, want empty env marker; env=%#v", key, envMap[key], envMap)
		}
	}

	cleanRuntimeRoot := filepath.Clean(runtimeRoot)
	if envMap["HOME"] != filepath.Join(cleanRuntimeRoot, "home") {
		t.Fatalf("HOME = %q, want isolated home", envMap["HOME"])
	}
	if envMap["TMPDIR"] != filepath.Join(cleanRuntimeRoot, "tmp") {
		t.Fatalf("TMPDIR = %q, want isolated tmp", envMap["TMPDIR"])
	}
	if envMap["GOENV"] != "off" {
		t.Fatalf("GOENV = %q, want off", envMap["GOENV"])
	}
	if envMap["GIT_CONFIG_GLOBAL"] != filepath.Join(cleanRuntimeRoot, "home", ".gitconfig") {
		t.Fatalf("GIT_CONFIG_GLOBAL = %q, want isolated git config", envMap["GIT_CONFIG_GLOBAL"])
	}
	if envMap["GIT_CONFIG_SYSTEM"] != filepath.Join(cleanRuntimeRoot, "home", ".config", "gitconfig-system") {
		t.Fatalf("GIT_CONFIG_SYSTEM = %q, want isolated system git config", envMap["GIT_CONFIG_SYSTEM"])
	}
	if envMap["NPM_CONFIG_PREFIX"] != filepath.Join(cleanRuntimeRoot, "npm-prefix") {
		t.Fatalf("NPM_CONFIG_PREFIX = %q, want isolated npm prefix", envMap["NPM_CONFIG_PREFIX"])
	}
	if envMap["CARGO_HOME"] != filepath.Join(cleanRuntimeRoot, "cargo-home") {
		t.Fatalf("CARGO_HOME = %q, want isolated cargo home", envMap["CARGO_HOME"])
	}
	if _, err := os.Stat(runtimeRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runtime root should be removed, stat error = %v", err)
	}
}

func TestHostReadOnlyExecutor_DoesNotInheritGitExternalDiff(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "changed\n")

	helperBin := filepath.Join(t.TempDir(), "helper-bin")
	marker := filepath.Join(t.TempDir(), "external-diff-ran")
	createProbeTestScriptCommand(t, helperBin, "external-diff", "touch "+shellSingleQuoteForProbeTest(marker))

	executor := newHostReadOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + os.Getenv("PATH"),
		"GIT_EXTERNAL_DIFF=" + filepath.Join(helperBin, "external-diff"),
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "host-readonly-git-external-diff",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 4 * 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"diff", "--", "keep.txt"}},
		},
	})

	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbePassed, result.Error, firstCommandOutput(result))
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GIT_EXTERNAL_DIFF helper should not run, marker stat error = %v", err)
	}
}

func TestHostReadOnlyExecutor_GoTestSeesHostModuleCacheReadOnly(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	hostModCache := filepath.Join(t.TempDir(), "go-mod")
	writeTestFile(t, filepath.Join(hostModCache, "example.com", "dep@v1.0.0", "marker.txt"), "module-cache\n")

	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createProbeTestScriptCommand(t, safeBin, "go", strings.Join([]string{
		`if [ "$1" != "test" ]; then echo "unexpected go command: $*" >&2; exit 2; fi`,
		`IFS= read -r marker < "$GOMODCACHE/example.com/dep@v1.0.0/marker.txt" || exit 2`,
		`printf '%s\n' "$marker"`,
		`if printf x > "$GOMODCACHE/write-check" 2>/dev/null; then echo "GOMODCACHE is writable" >&2; exit 3; fi`,
	}, "\n"))

	executor := newHostReadOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + safeBin,
		"GOMODCACHE=" + hostModCache,
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "host-readonly-go-mod-cache",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 4 * 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "./probe"}},
		},
	})

	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q output=%q)", result.Status, domain.ReviewProbePassed, result.Error, firstCommandOutput(result))
	}
	if !strings.Contains(firstCommandOutput(result), "module-cache") {
		t.Fatalf("output = %q, want host module cache marker", firstCommandOutput(result))
	}
	if _, err := os.Stat(filepath.Join(hostModCache, "write-check")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("host module cache should not be writable, stat error = %v", err)
	}
}

func TestHostReadOnlyExecutor_AppendsCleanupErrorWithoutChangingStatus(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createProbeTestScriptCommand(t, safeBin, "git", "exit 0")

	runtimeRoot := filepath.Join(t.TempDir(), "xelyon-review-host-readonly-cleanup-error")
	t.Cleanup(func() {
		_ = os.RemoveAll(runtimeRoot)
	})

	executor := newHostReadOnlyExecutor(repo)
	executor.baseEnv = []string{"PATH=" + safeBin}
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
			return "", err
		}
		return runtimeRoot, nil
	}
	executor.removeAll = func(path string) error {
		return errors.New("cleanup failed")
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:             "host-readonly-cleanup-error",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})

	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbePassed, result.Error)
	}
	if !strings.Contains(result.Error, "failed to remove host_readonly runtime root") {
		t.Fatalf("Error = %q, want cleanup error", result.Error)
	}
}

func TestProbeRunner_HostReadOnlyTimedOut(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-timeout",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        100 * time.Millisecond,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "TestProbeSleep"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeTimedOut {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbeTimedOut, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != domain.ReviewProbeTimedOut {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, domain.ReviewProbeTimedOut)
	}
}

func TestProbeRunner_HostReadOnlyOutputTruncated(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-truncate",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 32,
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"large.txt"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbePassed {
		t.Fatalf("Status = %q, want %q", result.Status, domain.ReviewProbePassed)
	}
	if !result.OutputTruncated {
		t.Fatal("OutputTruncated = false, want true")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !result.CommandResults[0].OutputTruncated {
		t.Fatal("CommandResults[0].OutputTruncated = false, want true")
	}
	if len(result.CommandResults[0].Output) > 32 {
		t.Fatalf("len(CommandResults[0].Output) = %d, want <= 32", len(result.CommandResults[0].Output))
	}
}

func TestProbeRunner_HostReadOnlyArgsDoNotUseShell(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)
	injectedPath := filepath.Join(repo, "shell_pwned")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-no-shell",
		Mode:           domain.ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short; touch shell_pwned"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q", result.Status, domain.ReviewProbeFailed)
	}
	if _, err := os.Stat(injectedPath); !os.IsNotExist(err) {
		t.Fatalf("shell-like argument should not create %s, stat error = %v", injectedPath, err)
	}
}

func parseProbeKeyValueOutput(t *testing.T, output string) map[string]string {
	t.Helper()

	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("output line %q is not key=value in output:\n%s", line, output)
		}
		values[key] = value
	}
	return values
}

func shellSingleQuoteForProbeTest(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

package review

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProbeGoModuleCacheReadOnlyBind_UsesExplicitHostCache(t *testing.T) {
	repo := t.TempDir()
	hostCache := filepath.Join(t.TempDir(), "go-mod")
	if err := os.MkdirAll(hostCache, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", hostCache, err)
	}
	target := filepath.Join(t.TempDir(), "runtime", "cache", "go-mod")

	bind, ok := probeGoModuleCacheReadOnlyBind([]string{"GOMODCACHE=" + hostCache}, repo, target)
	if !ok {
		t.Fatal("probeGoModuleCacheReadOnlyBind() ok = false, want true")
	}
	if bind.source != filepath.Clean(hostCache) || bind.target != filepath.Clean(target) {
		t.Fatalf("bind = %#v, want source %q target %q", bind, filepath.Clean(hostCache), filepath.Clean(target))
	}
}

func TestProbeGoModuleCacheReadOnlyBind_UsesDefaultHomeCache(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	hostCache := filepath.Join(home, "go", "pkg", "mod")
	if err := os.MkdirAll(hostCache, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", hostCache, err)
	}
	target := filepath.Join(t.TempDir(), "runtime", "cache", "go-mod")

	bind, ok := probeGoModuleCacheReadOnlyBind([]string{"HOME=" + home}, repo, target)
	if !ok {
		t.Fatal("probeGoModuleCacheReadOnlyBind() ok = false, want true")
	}
	if bind.source != filepath.Clean(hostCache) || bind.target != filepath.Clean(target) {
		t.Fatalf("bind = %#v, want source %q target %q", bind, filepath.Clean(hostCache), filepath.Clean(target))
	}
}

func TestProbeGoModuleCacheReadOnlyBind_IgnoresUnsafeOrUnavailableCaches(t *testing.T) {
	repo := t.TempDir()
	repoCache := filepath.Join(repo, "go", "pkg", "mod")
	if err := os.MkdirAll(repoCache, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", repoCache, err)
	}
	target := filepath.Join(t.TempDir(), "runtime", "cache", "go-mod")

	tests := []struct {
		name string
		env  []string
	}{
		{name: "missing explicit cache", env: []string{"GOMODCACHE=" + filepath.Join(t.TempDir(), "missing")}},
		{name: "relative explicit cache", env: []string{"GOMODCACHE=relative/pkg/mod"}},
		{name: "repo-contained explicit cache", env: []string{"GOMODCACHE=" + repoCache}},
		{name: "missing default home cache", env: []string{"HOME=" + t.TempDir()}},
	}

	if runtime.GOOS != "windows" {
		linkCache := filepath.Join(t.TempDir(), "linked-mod")
		if err := os.Symlink(repoCache, linkCache); err != nil {
			t.Fatalf("Symlink(%q, %q) error = %v", repoCache, linkCache, err)
		}
		tests = append(tests, struct {
			name string
			env  []string
		}{name: "symlink into repo cache", env: []string{"GOMODCACHE=" + linkCache}})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if bind, ok := probeGoModuleCacheReadOnlyBind(tt.env, repo, target); ok {
				t.Fatalf("probeGoModuleCacheReadOnlyBind() = %#v, true; want no bind", bind)
			}
		})
	}
}

func TestProbeProcessSandbox_BindsWritableRootsBeforeReadOnlyModuleCache(t *testing.T) {
	runtimeRoot := filepath.Join(t.TempDir(), "runtime")
	hostCache := filepath.Join(t.TempDir(), "go-mod")
	targetCache := filepath.Join(runtimeRoot, "cache", "go-mod")
	sandbox := probeProcessSandbox{
		enabled:    true,
		runnerPath: "/usr/bin/bwrap",
		readOnlyBinds: []probeProcessSandboxBind{
			{source: hostCache, target: targetCache},
		},
		readWriteBinds: []probeProcessSandboxBind{
			{source: runtimeRoot, target: runtimeRoot},
		},
	}

	args, err := sandbox.buildBubblewrapArgs(probeExecCommand{
		commandPath: "/bin/sh",
		workDir:     runtimeRoot,
	})
	if err != nil {
		t.Fatalf("buildBubblewrapArgs() error = %v", err)
	}

	bindIndex := indexProbeSandboxArgTriple(args, "--bind", filepath.Clean(runtimeRoot), filepath.Clean(runtimeRoot))
	roBindIndex := indexProbeSandboxArgTriple(args, "--ro-bind", filepath.Clean(hostCache), filepath.Clean(targetCache))
	if bindIndex < 0 || roBindIndex < 0 {
		t.Fatalf("bubblewrap args missing module cache bind order markers: %s", strings.Join(args, " "))
	}
	if bindIndex > roBindIndex {
		t.Fatalf("runtime bind index = %d, module cache ro-bind index = %d; want writable parent first", bindIndex, roBindIndex)
	}
}

func indexProbeSandboxArgTriple(args []string, flag, source, target string) int {
	for i := 0; i+2 < len(args); i++ {
		if args[i] == flag && args[i+1] == source && args[i+2] == target {
			return i
		}
	}
	return -1
}

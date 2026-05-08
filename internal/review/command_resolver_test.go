package review

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveCommandPath_AllowsExecutableFromSafePath(t *testing.T) {
	repoRoot := t.TempDir()
	safeBin := filepath.Join(t.TempDir(), "bin")
	executablePath := createCommandResolverTestExecutable(t, safeBin, "tool")

	resolved, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env:      []string{"PATH=" + safeBin},
	})
	if err != nil {
		t.Fatalf("resolveCommandPath() error = %v", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(executablePath) {
		t.Fatalf("resolved = %q, want %q", resolved, executablePath)
	}
}

func TestResolveCommandPath_BlocksCommandPathInput(t *testing.T) {
	repoRoot := t.TempDir()
	safeBin := filepath.Join(t.TempDir(), "bin")
	createCommandResolverTestExecutable(t, safeBin, "tool")

	tests := []string{"./tool", "../tool", "/tmp/tool", `.\tool`, `C:\tool.exe`}
	for _, command := range tests {
		t.Run(command, func(t *testing.T) {
			_, err := resolveCommandPath(command, commandResolutionContext{
				RepoRoot: repoRoot,
				WorkDir:  repoRoot,
				Env:      []string{"PATH=" + safeBin},
			})
			if err == nil {
				t.Fatalf("resolveCommandPath(%q) error = nil", command)
			}
			if !errors.Is(err, ErrHostReadOnlyBlocked) {
				t.Fatalf("resolveCommandPath(%q) error = %v, want ErrHostReadOnlyBlocked", command, err)
			}
		})
	}
}

func TestResolveCommandPath_BlocksWhenPATHMissingOrNotFound(t *testing.T) {
	repoRoot := t.TempDir()
	tests := []struct {
		name string
		env  []string
	}{
		{name: "empty path", env: []string{"PATH="}},
		{name: "path missing", env: []string{"LANG=C"}},
		{name: "not found", env: []string{"PATH=" + t.TempDir()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveCommandPath("tool", commandResolutionContext{
				RepoRoot: repoRoot,
				WorkDir:  repoRoot,
				Env:      tt.env,
			})
			if err == nil {
				t.Fatalf("resolveCommandPath() error = nil")
			}
			if !errors.Is(err, ErrHostReadOnlyBlocked) {
				t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
			}
		})
	}
}

func TestResolveCommandPath_BlocksExecutableInsideRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	repoBin := filepath.Join(repoRoot, "bin")
	createCommandResolverTestExecutable(t, repoBin, "tool")

	_, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env:      []string{"PATH=" + repoBin},
	})
	if err == nil {
		t.Fatal("resolveCommandPath() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestResolveCommandPath_UsesWorkDirForRelativePathEntries(t *testing.T) {
	repoRoot := t.TempDir()
	cwdRoot := t.TempDir()
	workDir := t.TempDir()
	cwdBin := filepath.Join(cwdRoot, "bin")
	workBin := filepath.Join(workDir, "bin")

	cwdTool := createCommandResolverTestExecutable(t, cwdBin, "tool")
	workTool := createCommandResolverTestExecutable(t, workBin, "tool")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(cwdRoot); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", cwdRoot, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	resolved, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  workDir,
		Env:      []string{"PATH=bin"},
	})
	if err != nil {
		t.Fatalf("resolveCommandPath() error = %v", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(workTool) {
		t.Fatalf("resolved = %q, want workdir executable %q (cwd executable=%q)", resolved, workTool, cwdTool)
	}
}

func TestResolveCommandPath_BlocksSymlinkedExecutableToRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	repoBin := filepath.Join(repoRoot, "bin")
	repoExecutable := createCommandResolverTestExecutable(t, repoBin, "tool")

	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createCommandResolverTestExecutableSymlink(t, repoExecutable, filepath.Join(safeBin, "tool"))

	_, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env:      []string{"PATH=" + safeBin},
	})
	if err == nil {
		t.Fatal("resolveCommandPath() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestResolveCommandPath_BlocksPathDirectorySymlinkToRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	repoBin := filepath.Join(repoRoot, "bin")
	createCommandResolverTestExecutable(t, repoBin, "tool")

	safeRoot := t.TempDir()
	symlinkedBin := filepath.Join(safeRoot, "safe-bin")
	createCommandResolverTestExecutableSymlink(t, repoBin, symlinkedBin)

	_, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env:      []string{"PATH=" + symlinkedBin},
	})
	if err == nil {
		t.Fatal("resolveCommandPath() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestResolveCommandPath_BlocksExecutableInsideScratchDir(t *testing.T) {
	repoRoot := t.TempDir()
	scratchDir := t.TempDir()
	scratchBin := filepath.Join(scratchDir, "bin")
	createCommandResolverTestExecutable(t, scratchBin, "tool")

	_, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot:   repoRoot,
		ScratchDir: scratchDir,
		WorkDir:    repoRoot,
		Env:        []string{"PATH=" + scratchBin},
	})
	if err == nil {
		t.Fatal("resolveCommandPath() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestResolveCommandPath_BlocksSymlinkedExecutableToScratchDir(t *testing.T) {
	repoRoot := t.TempDir()
	scratchDir := t.TempDir()
	scratchBin := filepath.Join(scratchDir, "bin")
	scratchExecutable := createCommandResolverTestExecutable(t, scratchBin, "tool")

	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createCommandResolverTestExecutableSymlink(t, scratchExecutable, filepath.Join(safeBin, "tool"))

	_, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot:   repoRoot,
		ScratchDir: scratchDir,
		WorkDir:    repoRoot,
		Env:        []string{"PATH=" + safeBin},
	})
	if err == nil {
		t.Fatal("resolveCommandPath() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("resolveCommandPath() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func createCommandResolverTestExecutable(t *testing.T, dir, base string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}

	name := base
	content := "#!/bin/sh\nexit 0\n"
	mode := os.FileMode(0o755)

	if runtime.GOOS == "windows" {
		name = base + ".cmd"
		content = "@echo off\r\nexit /b 0\r\n"
		mode = 0o644
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", path, err)
	}
	return filepath.Clean(absPath)
}

func createCommandResolverTestExecutableSymlink(t *testing.T, target, linkPath string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(linkPath), err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("os.Symlink(%q, %q) is not supported in this environment: %v", target, linkPath, err)
	}
}

func TestResolveCommandPath_PathLookupIsOrdered(t *testing.T) {
	repoRoot := t.TempDir()
	firstBin := filepath.Join(t.TempDir(), "first")
	secondBin := filepath.Join(t.TempDir(), "second")

	first := createCommandResolverTestExecutable(t, firstBin, "tool")
	second := createCommandResolverTestExecutable(t, secondBin, "tool")

	resolved, err := resolveCommandPath("tool", commandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env: []string{
			"PATH=" + strings.Join([]string{firstBin, secondBin}, string(os.PathListSeparator)),
		},
	})
	if err != nil {
		t.Fatalf("resolveCommandPath() error = %v", err)
	}
	if filepath.Clean(resolved) != filepath.Clean(first) {
		t.Fatalf("resolved = %q, want first PATH executable %q (second=%q)", resolved, first, second)
	}
}

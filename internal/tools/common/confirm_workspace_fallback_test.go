package common

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }

func TestIsInsideWorkspace_FallbackPaths(t *testing.T) {
	t.Run("broken file symlink is treated as outside", func(t *testing.T) {
		workspace := t.TempDir()
		broken := filepath.Join(workspace, "broken.txt")
		if err := os.Symlink(filepath.Join(workspace, "missing.txt"), broken); err != nil {
			t.Skipf("symlink creation failed: %v", err)
		}

		origDir, _ := os.Getwd()
		if err := os.Chdir(workspace); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if isInsideWorkspace(broken) {
			t.Fatalf("isInsideWorkspace(%q) = true, want false for broken symlink", broken)
		}
	})

	t.Run("broken parent symlink is treated as outside", func(t *testing.T) {
		workspace := t.TempDir()
		parent := filepath.Join(workspace, "broken-parent")
		if err := os.Symlink(filepath.Join(workspace, "missing-dir"), parent); err != nil {
			t.Skipf("symlink creation failed: %v", err)
		}

		origDir, _ := os.Getwd()
		if err := os.Chdir(workspace); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		target := filepath.Join(parent, "new.txt")
		if isInsideWorkspace(target) {
			t.Fatalf("isInsideWorkspace(%q) = true, want false for broken parent symlink", target)
		}
	})

	t.Run("missing nested parent falls back to abs path comparison", func(t *testing.T) {
		workspace := t.TempDir()

		origDir, _ := os.Getwd()
		if err := os.Chdir(workspace); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if !isInsideWorkspace(filepath.Join("new-dir", "nested", "file.txt")) {
			t.Fatal("nested missing path under workspace should be treated as inside")
		}
	})

	t.Run("getwd failure falls back to inside", func(t *testing.T) {
		workspace := t.TempDir()

		origDir, _ := os.Getwd()
		if err := os.Chdir(workspace); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(workspace); err != nil {
			_ = os.Chdir(origDir)
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		if !isInsideWorkspace("/etc/hosts") {
			t.Fatal("Getwd failure should fall back to inside=true")
		}
	})
}

func TestIsInsideWorkspaceWithOps_ErrorFallbacks(t *testing.T) {
	t.Run("abs failure falls back to inside", func(t *testing.T) {
		got := isInsideWorkspaceWithOps("relative.txt", workspacePathOps{
			abs: func(string) (string, error) { return "", errors.New("abs failed") },
		})
		if !got {
			t.Fatal("abs failure should fall back to inside=true")
		}
	})

	t.Run("getwd failure falls back to inside", func(t *testing.T) {
		got := isInsideWorkspaceWithOps("/workspace/file.txt", workspacePathOps{
			abs:   func(path string) (string, error) { return path, nil },
			getwd: func() (string, error) { return "", errors.New("getwd failed") },
		})
		if !got {
			t.Fatal("getwd failure should fall back to inside=true")
		}
	})

	t.Run("workspace abs failure falls back to inside", func(t *testing.T) {
		callCount := 0
		got := isInsideWorkspaceWithOps("relative.txt", workspacePathOps{
			abs: func(path string) (string, error) {
				callCount++
				if callCount == 1 {
					return "/workspace/file.txt", nil
				}
				return "", errors.New("workspace abs failed")
			},
			getwd: func() (string, error) { return "/workspace", nil },
		})
		if !got {
			t.Fatal("workspace abs failure should fall back to inside=true")
		}
	})

	t.Run("workspace eval symlink failure falls back to raw workspace path", func(t *testing.T) {
		got := isInsideWorkspaceWithOps("/workspace/file.txt", workspacePathOps{
			abs:   func(path string) (string, error) { return path, nil },
			getwd: func() (string, error) { return "/workspace", nil },
			evalSymlinks: func(path string) (string, error) {
				if path == "/workspace" {
					return "", errors.New("workspace eval failed")
				}
				return path, nil
			},
			lstat: func(path string) (os.FileInfo, error) { return fakeFileInfo{}, nil },
		})
		if !got {
			t.Fatal("workspace eval symlink failure should still compare with raw workspace path")
		}
	})
}

func TestAllPathsInsideWorkspace_TargetPaths(t *testing.T) {
	if AllPathsInsideWorkspace(ToolConfirmContext{TargetPath: "/etc/hosts"}) {
		t.Fatal("AllPathsInsideWorkspace() = true, want false when target path escapes workspace")
	}

	if !AllPathsInsideWorkspace(ToolConfirmContext{
		TargetPaths: []string{"internal/config/config.go", "internal/tools/common/confirm.go"},
	}) {
		t.Fatal("AllPathsInsideWorkspace() = false, want true for all target paths inside workspace")
	}

	if AllPathsInsideWorkspace(ToolConfirmContext{
		TargetPaths: []string{"internal/config/config.go", "/etc/hosts"},
	}) {
		t.Fatal("AllPathsInsideWorkspace() = true, want false when any target path escapes workspace")
	}
}

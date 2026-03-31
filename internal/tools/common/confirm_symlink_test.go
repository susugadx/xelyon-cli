package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInsideWorkspace_SymlinkEscape(t *testing.T) {
	// workspace 内から workspace 外を指す symlink
	workspace := t.TempDir()
	outside := t.TempDir()

	// outside にファイルを作成
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// workspace 内に outside を指す symlink を作成
	link := filepath.Join(workspace, "escape_link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation failed (permission): %v", err)
	}

	// CWD を workspace に変更
	origDir, _ := os.Getwd()
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// symlink 経由のパスは workspace 外として検出されるべき
	escapePath := filepath.Join(link, "secret.txt")
	if isInsideWorkspace(escapePath) {
		t.Errorf("isInsideWorkspace(%q) = true, want false (symlink to %q)", escapePath, outside)
	}

	// symlink ディレクトリ自体も workspace 外
	if isInsideWorkspace(link) {
		t.Errorf("isInsideWorkspace(%q) = true, want false (symlink to %q)", link, outside)
	}
}

func TestIsInsideWorkspace_SymlinkCWD(t *testing.T) {
	// P1 修正検証: symlink 経由で CWD に入った場合、
	// symlink ベースの絶対パスでも workspace 内ファイルを正しく inside と判定する
	realWorkspace := t.TempDir()
	subdir := filepath.Join(realWorkspace, "src")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(subdir, "main.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// symlink 経由の CWD を作成
	symlinkParent := t.TempDir()
	symlinkCWD := filepath.Join(symlinkParent, "workspace_link")
	if err := os.Symlink(realWorkspace, symlinkCWD); err != nil {
		t.Skipf("symlink creation failed: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(symlinkCWD); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// symlink ベースの絶対パスで workspace 内ファイルを指定
	symlinkBasedPath := filepath.Join(symlinkCWD, "src", "main.go")
	if !isInsideWorkspace(symlinkBasedPath) {
		t.Errorf("isInsideWorkspace(%q) = false, want true (symlink CWD, file is inside)", symlinkBasedPath)
	}

	// 実体パスでも workspace 内
	realPath := filepath.Join(realWorkspace, "src", "main.go")
	if !isInsideWorkspace(realPath) {
		t.Errorf("isInsideWorkspace(%q) = false, want true (real path, file is inside)", realPath)
	}

	// 相対パスでも workspace 内
	if !isInsideWorkspace("src/main.go") {
		t.Error("isInsideWorkspace(\"src/main.go\") = false, want true (relative path)")
	}
}

func TestIsInsideWorkspace_SymlinkCWD_EscapeStillBlocked(t *testing.T) {
	// symlink 経由 CWD でも、workspace 外への symlink escape はブロックされる
	realWorkspace := t.TempDir()
	outside := t.TempDir()

	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// workspace 内に outside を指す symlink
	escapeLink := filepath.Join(realWorkspace, "escape")
	if err := os.Symlink(outside, escapeLink); err != nil {
		t.Skipf("symlink creation failed: %v", err)
	}

	// symlink 経由の CWD
	symlinkParent := t.TempDir()
	symlinkCWD := filepath.Join(symlinkParent, "ws_link")
	if err := os.Symlink(realWorkspace, symlinkCWD); err != nil {
		t.Skipf("symlink creation failed: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(symlinkCWD); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// escape symlink 経由のパスは workspace 外
	escapePath := filepath.Join(symlinkCWD, "escape", "secret.txt")
	if isInsideWorkspace(escapePath) {
		t.Errorf("isInsideWorkspace(%q) = true, want false (symlink escape to %q)", escapePath, outside)
	}
}

func TestIsInsideWorkspace_NonSymlink(t *testing.T) {
	workspace := t.TempDir()
	subdir := filepath.Join(workspace, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(subdir, "file.go")
	if err := os.WriteFile(file, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if !isInsideWorkspace(file) {
		t.Errorf("isInsideWorkspace(%q) = false, want true (real path inside workspace)", file)
	}
}

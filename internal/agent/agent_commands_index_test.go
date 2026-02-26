package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

func TestGatherIndexFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// テスト用のファイルを作成
	files := []string{
		"main.go",
		"app.py",
		"README.md",
		"config.yaml",
		"image.jpg",
		"photo.png",
		"video.mp4",
		"Makefile",
		"Dockerfile",
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f, err)
		}
	}

	extensions := []string{".go", ".py", ".md", ".yaml", "Makefile", "Dockerfile"}
	gathered := gatherIndexFiles(tmpDir, extensions)

	// パスをベース名のみに変換してソート
	var basenames []string
	for _, p := range gathered {
		basenames = append(basenames, filepath.Base(p))
	}
	sort.Strings(basenames)

	expected := []string{
		"Dockerfile",
		"Makefile",
		"README.md",
		"app.py",
		"config.yaml",
		"main.go",
	}
	sort.Strings(expected)

	if len(basenames) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(basenames), basenames)
	}

	for i, v := range basenames {
		if v != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], v)
		}
	}
}

func TestGatherIndexFiles_GitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("git is not available")
	}

	// ユーザー情報の設定 (CI環境などでgit commitが失敗するのを防ぐ)
	_ = exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()

	// ファイル作成
	trackedFile := "main.go"
	untrackedFile := "test.go"

	_ = os.WriteFile(filepath.Join(tmpDir, trackedFile), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, untrackedFile), []byte("package test"), 0644)

	// git add
	cmd = exec.Command("git", "add", trackedFile)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	extensions := []string{".go"}
	gathered := gatherIndexFiles(tmpDir, extensions)

	if len(gathered) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(gathered), gathered)
	}

	if filepath.Base(gathered[0]) != trackedFile {
		t.Errorf("expected %s, got %s", trackedFile, filepath.Base(gathered[0]))
	}
}

func TestGatherIndexFiles_GitRepo_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()

	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("git is not available")
	}

	// ファイル作成のみ、git add なし
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	extensions := []string{".go"}
	gathered := gatherIndexFiles(tmpDir, extensions)

	if len(gathered) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(gathered), gathered)
	}
}

func TestIsValidIndexFile(t *testing.T) {
	extensions := []string{".go", ".py", ".ts", ".md", ".yaml", "Makefile", "Dockerfile"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// 対象
		{"Go file", "main.go", true},
		{"Python file", "app.py", true},
		{"TypeScript file", "index.ts", true},
		{"Markdown file", "README.md", true},
		{"YAML file", "config.yaml", true},
		{"Makefile", "Makefile", true},
		{"Dockerfile", "Dockerfile", true},

		// 除外 (一般)
		{"go.sum", "go.sum", false},
		{"go.mod", "go.mod", false},
		{"package-lock.json", "package-lock.json", false},
		{"yarn.lock", "yarn.lock", false},
		{"pnpm-lock.yaml", "pnpm-lock.yaml", false},

		// 除外 (.min. / .generated.)
		{"Minified JS", "app.min.js", false},
		{"Minified CSS", "style.min.css", false},
		{"Generated Go", "api.generated.go", false},

		// 対象外 (拡張子)
		{"Image PNG", "image.png", false},
		{"Video MP4", "video.mp4", false},
		{"Archive ZIP", "archive.zip", false},
		{"Binary EXE", "binary.exe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIndexFile(tt.path, extensions)
			if result != tt.expected {
				t.Errorf("isValidIndexFile(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

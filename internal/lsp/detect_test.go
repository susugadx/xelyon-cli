package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectLanguages(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()

	// テスト用のファイルを作成
	files := []string{
		"main.go",
		"internal/api/client.go",
		"internal/api/types.go",
		"src/index.ts",
		"src/components/Button.tsx",
		"scripts/deploy.py",
		"README.md", // 対象外
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("// test file"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// 検出実行
	languages, err := DetectProjectLanguages(tmpDir)
	if err != nil {
		t.Fatalf("DetectProjectLanguages failed: %v", err)
	}

	// 期待する言語
	expected := map[string]int{
		"go":         3, // main.go, client.go, types.go
		"typescript": 2, // index.ts, Button.tsx
		"python":     1, // deploy.py
		"markdown":   1, // README.md (now detected with LSP support)
	}

	// 検証
	found := make(map[string]int)
	for _, info := range languages {
		found[info.ServerKey] = info.FileCount
	}

	for lang, expectedCount := range expected {
		if count, ok := found[lang]; !ok {
			t.Errorf("Expected language %s to be detected", lang)
		} else if count != expectedCount {
			t.Errorf("Expected %d files for %s, got %d", expectedCount, lang, count)
		}
	}
}

func TestDetectProjectLanguages_SkipDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// スキップ対象のディレクトリにファイルを作成
	skipFiles := []string{
		"node_modules/lodash/index.js",
		"vendor/github.com/pkg/errors/errors.go",
		".git/objects/pack/pack.go",
		"__pycache__/cache.py",
	}

	for _, f := range skipFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("// test file"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// 対象のファイルも作成
	targetFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(targetFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	languages, err := DetectProjectLanguages(tmpDir)
	if err != nil {
		t.Fatalf("DetectProjectLanguages failed: %v", err)
	}

	// Go のみ検出されるはず（1ファイル）
	if len(languages) != 1 {
		t.Errorf("Expected 1 language, got %d", len(languages))
	}

	if len(languages) > 0 && languages[0].ServerKey != "go" {
		t.Errorf("Expected go, got %s", languages[0].ServerKey)
	}

	if len(languages) > 0 && languages[0].FileCount != 1 {
		t.Errorf("Expected 1 file, got %d", languages[0].FileCount)
	}
}

func TestDetectProjectLanguages_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	languages, err := DetectProjectLanguages(tmpDir)
	if err != nil {
		t.Fatalf("DetectProjectLanguages failed: %v", err)
	}

	if len(languages) != 0 {
		t.Errorf("Expected 0 languages, got %d", len(languages))
	}
}

func TestGetLanguageExtensions(t *testing.T) {
	tests := []struct {
		serverKey string
		expected  []string
	}{
		{"go", []string{".go"}},
		{"typescript", []string{".js", ".jsx", ".ts", ".tsx"}},
		{"python", []string{".py"}},
		{"rust", []string{".rs"}},
		// New languages
		{"java", []string{".java"}},
		{"ruby", []string{".rb"}},
		{"php", []string{".php"}},
		{"yaml", []string{".yaml", ".yml"}},
		{"markdown", []string{".markdown", ".md"}},
		{"bash", []string{".bash", ".sh", ".zsh"}},
		{"unknown", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.serverKey, func(t *testing.T) {
			exts := GetLanguageExtensions(tt.serverKey)
			if len(exts) != len(tt.expected) {
				t.Errorf("Expected %d extensions, got %d", len(tt.expected), len(exts))
				return
			}
			for i, ext := range exts {
				if ext != tt.expected[i] {
					t.Errorf("Expected extension %s, got %s", tt.expected[i], ext)
				}
			}
		})
	}
}

func TestDetectProjectLanguages_NewLanguages(t *testing.T) {
	// Test detection of newly added languages
	tmpDir := t.TempDir()

	// Create test files for new languages
	files := []string{
		"Main.java",
		"style.css",
		"index.html",
		"config.yaml",
		"deploy.sh",
		"notes.md",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("// test file"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	languages, err := DetectProjectLanguages(tmpDir)
	if err != nil {
		t.Fatalf("DetectProjectLanguages failed: %v", err)
	}

	// Expected languages
	expected := map[string]bool{
		"java":     true,
		"css":      true,
		"html":     true,
		"yaml":     true,
		"bash":     true,
		"markdown": true,
	}

	found := make(map[string]bool)
	for _, info := range languages {
		found[info.ServerKey] = true
	}

	for lang := range expected {
		if !found[lang] {
			t.Errorf("Expected language %s to be detected", lang)
		}
	}
}

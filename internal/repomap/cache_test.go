//go:build !norepomap
// +build !norepomap

package repomap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cache := &RepoMapCache{
		RootPath:  tmpDir,
		UpdatedAt: time.Now(),
		Files: map[string]*CachedFile{
			filepath.Join(tmpDir, "test.go"): {
				Path:    filepath.Join(tmpDir, "test.go"),
				ModTime: time.Now().Truncate(time.Second), // ファイルシステム精度
				Symbols: []Symbol{{Name: "TestFunc", Kind: "function", Line: 1}},
			},
		},
	}

	// 保存
	err := SaveCache(cache)
	if err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	// 読み込み
	loaded, err := LoadCache(tmpDir)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if len(loaded.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(loaded.Files))
	}

	// シンボル確認
	cachedFile := loaded.Files[filepath.Join(tmpDir, "test.go")]
	if cachedFile == nil {
		t.Fatal("Expected cached file not found")
	}
	if len(cachedFile.Symbols) != 1 {
		t.Errorf("Expected 1 symbol, got %d", len(cachedFile.Symbols))
	}
	if cachedFile.Symbols[0].Name != "TestFunc" {
		t.Errorf("Expected symbol name 'TestFunc', got '%s'", cachedFile.Symbols[0].Name)
	}
}

func TestLoadCacheNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// キャッシュが存在しない場合
	_, err := LoadCache(tmpDir)
	if err == nil {
		t.Error("Expected error for non-existent cache, got nil")
	}
}

func TestGetCachePath(t *testing.T) {
	path1, err := getCachePath("/some/project/path")
	if err != nil {
		t.Fatalf("getCachePath failed: %v", err)
	}

	path2, err := getCachePath("/another/project")
	if err != nil {
		t.Fatalf("getCachePath failed: %v", err)
	}

	// 異なるパスは異なるキャッシュファイル
	if path1 == path2 {
		t.Error("Different root paths should produce different cache paths")
	}

	// 同じパスは同じキャッシュファイル
	path3, _ := getCachePath("/some/project/path")
	if path1 != path3 {
		t.Error("Same root path should produce same cache path")
	}
}

func TestParallelBuild(t *testing.T) {
	// テスト用ディレクトリ作成
	tmpDir := t.TempDir()

	// テストファイル作成
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func Hello() {}
func World() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Build 実行
	rm := NewRepoMap(tmpDir, 0)
	err := rm.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 結果確認
	if len(rm.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(rm.Files))
	}

	// シンボル数確認（Hello + World）
	if len(rm.Files) > 0 && len(rm.Files[0].Symbols) < 2 {
		t.Errorf("Expected at least 2 symbols, got %d", len(rm.Files[0].Symbols))
	}
}

func TestIncrementalBuild(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイル作成
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main
func Hello() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 1回目の Build（キャッシュ生成）
	rm1 := NewRepoMap(tmpDir, 0)
	if err := rm1.Build(); err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	// 2回目の Build（キャッシュ使用）
	rm2 := NewRepoMap(tmpDir, 0)
	start := time.Now()
	if err := rm2.Build(); err != nil {
		t.Fatalf("Second build failed: %v", err)
	}
	elapsed := time.Since(start)

	// キャッシュ使用時は高速なはず（参考ログ）
	t.Logf("Second build took: %v", elapsed)

	if len(rm2.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(rm2.Files))
	}
}

func TestDeletedFileNotInResult(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイル作成
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main
func Hello() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 1回目の Build
	rm1 := NewRepoMap(tmpDir, 0)
	if err := rm1.Build(); err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	if len(rm1.Files) != 1 {
		t.Errorf("Expected 1 file after first build, got %d", len(rm1.Files))
	}

	// ファイル削除
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	// 2回目の Build（削除されたファイルは結果に含まれない）
	rm2 := NewRepoMap(tmpDir, 0)
	if err := rm2.Build(); err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	if len(rm2.Files) != 0 {
		t.Errorf("Expected 0 files after deletion, got %d", len(rm2.Files))
	}
}

func TestCacheWithModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイル作成
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main
func Hello() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 1回目の Build（キャッシュ生成）
	rm1 := NewRepoMap(tmpDir, 0)
	if err := rm1.Build(); err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	// 少し待ってからファイル更新（ModTime を変更するため）
	time.Sleep(10 * time.Millisecond)

	// ファイル変更
	newContent := `package main
func Hello() {}
func World() {}
func Foo() {}
`
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// 2回目の Build（変更ファイルは再解析される）
	rm2 := NewRepoMap(tmpDir, 0)
	if err := rm2.Build(); err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	if len(rm2.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(rm2.Files))
	}

	// 新しいシンボル数を確認（Hello + World + Foo = 3）
	if len(rm2.Files) > 0 && len(rm2.Files[0].Symbols) < 3 {
		t.Errorf("Expected at least 3 symbols after modification, got %d", len(rm2.Files[0].Symbols))
	}
}

func TestGetGitChangedFiles(t *testing.T) {
	// git リポジトリではないディレクトリの場合
	tmpDir := t.TempDir()
	result := getGitChangedFiles(tmpDir)
	if result != nil {
		t.Error("Expected nil for non-git directory")
	}
}

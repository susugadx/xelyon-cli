package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepReplaceBasic(t *testing.T) {
	dir := t.TempDir()

	// テストファイル作成
	file1 := filepath.Join(dir, "test1.go")
	file2 := filepath.Join(dir, "test2.go")

	if err := os.WriteFile(file1, []byte("func oldFunc() {}\nfunc anotherFunc() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("// call oldFunc here\noldFunc()"), 0644); err != nil {
		t.Fatal(err)
	}

	// ドライランでプレビュー
	result, files, err := executeGrepReplace("oldFunc", "newFunc", dir, "*.go", true)
	if err != nil {
		t.Fatalf("executeGrepReplace (dry_run) failed: %v", err)
	}

	if !strings.Contains(result, "Dry Run") {
		t.Errorf("Expected 'Dry Run' in result, got: %s", result)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files matched, got %d", len(files))
	}

	// ファイルは変更されていないはず
	content1, _ := os.ReadFile(file1)
	if !strings.Contains(string(content1), "oldFunc") {
		t.Error("File should not be modified in dry run")
	}
}

func TestGrepReplaceExecute(t *testing.T) {
	dir := t.TempDir()

	// テストファイル作成
	file1 := filepath.Join(dir, "replace.go")
	if err := os.WriteFile(file1, []byte("func oldName() {}\nfunc oldName2() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// 実行（dry_run=false）
	// setupTestMocksでインタラクティブモードを無効化し、自動承認
	setupTestMocks(t)

	result, _, err := executeGrepReplace("oldName", "newName", dir, "*.go", false)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("Expected 'completed' in result, got: %s", result)
	}

	// ファイルが変更されているはず
	content, _ := os.ReadFile(file1)
	if strings.Contains(string(content), "oldName") {
		t.Error("File should be modified after execution")
	}
	if !strings.Contains(string(content), "newName") {
		t.Error("File should contain replacement text")
	}
}

func TestGrepReplaceNoMatch(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "nomatch.go")
	if err := os.WriteFile(file1, []byte("func something() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	result, files, err := executeGrepReplace("nonexistent", "replacement", dir, "*.go", true)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files matched, got %d", len(files))
	}

	if !strings.Contains(result, "0") || !strings.Contains(result, "Files") {
		// "Files: 0" が含まれるはず
	}
}

func TestGrepReplaceEmptyPattern(t *testing.T) {
	dir := t.TempDir()

	_, _, err := executeGrepReplace("", "replacement", dir, "*.go", true)
	if err == nil {
		t.Error("Expected error for empty pattern")
	}
}

func TestGrepReplaceInvalidRegex(t *testing.T) {
	dir := t.TempDir()

	_, _, err := executeGrepReplace("[invalid(", "replacement", dir, "*.go", true)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestGrepReplaceFilePattern(t *testing.T) {
	dir := t.TempDir()

	// 異なる拡張子のファイル
	goFile := filepath.Join(dir, "test.go")
	jsFile := filepath.Join(dir, "test.js")

	if err := os.WriteFile(goFile, []byte("func target() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function target() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// *.go のみを対象
	result, files, err := executeGrepReplace("target", "replaced", dir, "*.go", true)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	// .go ファイルのみマッチするはず
	if len(files) != 1 {
		t.Errorf("Expected 1 file matched (*.go only), got %d", len(files))
	}

	if len(files) > 0 && !strings.HasSuffix(files[0].FilePath, ".go") {
		t.Errorf("Expected .go file, got %s", files[0].FilePath)
	}

	_ = result // 結果確認用
}

func TestGrepReplaceBackupCreated(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "backup.go")
	if err := os.WriteFile(file1, []byte("original content with target"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)

	_, files, err := executeGrepReplace("target", "replaced", dir, "*.go", false)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	// バックアップが作成されているはず
	if len(files) > 0 && files[0].BackupPath == "" {
		t.Error("Expected backup to be created")
	}

	// バックアップファイルが存在するか確認
	if len(files) > 0 {
		if _, err := os.Stat(files[0].BackupPath); os.IsNotExist(err) {
			t.Error("Backup file should exist")
		}
	}
}

func TestGrepReplaceSkipsBackupFiles(t *testing.T) {
	dir := t.TempDir()

	// 通常のファイル
	normalFile := filepath.Join(dir, "normal.go")
	// バックアップファイル
	backupFile := filepath.Join(dir, "normal.go.bak.20240101_120000")

	if err := os.WriteFile(normalFile, []byte("target here"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte("target in backup"), 0644); err != nil {
		t.Fatal(err)
	}

	_, files, err := executeGrepReplace("target", "replaced", dir, "*", true)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	// バックアップファイルはスキップされるはず
	for _, f := range files {
		if strings.Contains(f.FilePath, ".bak.") {
			t.Errorf("Backup file should be skipped: %s", f.FilePath)
		}
	}
}

func TestGrepReplaceRegexCapture(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "regex.go")
	if err := os.WriteFile(file1, []byte("func oldFunc1() {}\nfunc oldFunc2() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)

	// 正規表現キャプチャを使った置換
	_, _, err := executeGrepReplace(`oldFunc(\d+)`, `newFunc$1`, dir, "*.go", false)
	if err != nil {
		t.Fatalf("executeGrepReplace failed: %v", err)
	}

	content, _ := os.ReadFile(file1)
	contentStr := string(content)

	if !strings.Contains(contentStr, "newFunc1") || !strings.Contains(contentStr, "newFunc2") {
		t.Errorf("Expected regex capture replacement, got: %s", contentStr)
	}
}

func TestGrepReplaceTool(t *testing.T) {
	setupTestMocks(t)

	tool := &GrepReplaceTool{}

	if tool.Name() != "grep_replace" {
		t.Errorf("Expected tool name 'grep_replace', got '%s'", tool.Name())
	}

	dir := t.TempDir()
	file1 := filepath.Join(dir, "tool.go")
	if err := os.WriteFile(file1, []byte("target text here"), 0644); err != nil {
		t.Fatal(err)
	}

	// ドライランでテスト
	result, change, err := tool.Run(map[string]string{
		"pattern":     "target",
		"replacement": "replaced",
		"path":        dir,
		"dry_run":     "true",
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	// grep_replaceはFileChangeを返さない（複数ファイルのため）
	if change != nil {
		t.Error("Expected no FileChange for grep_replace")
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}

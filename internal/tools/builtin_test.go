package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// getToolFromRegistry はレジストリからツールを取得するヘルパー
func getToolFromRegistry(name string) Tool {
	return DefaultRegistry.GetTool(name)
}

// ===== Registry-based Tool Tests =====

func TestRegisteredTools(t *testing.T) {
	expectedTools := []string{
		"bash", "read_file", "write_file", "str_replace", "list_dir",
		"git_status", "git_diff", "git_add", "git_commit", "git_push",
		"git_log", "git_branch", "git_stash", "git_checkout",
		"search_code", "search_file", "web_search", "create_dir", "run_test",
		"append_file", "prepend_file", "format", "insert_after", "insert_before",
		"copy_file", "delete_lines", "delete_file", "move_file", "lint",
		"grep_replace", "http_request", "diff_files", "restore_backup",
		"list_backups", "ast_grep",
	}

	for _, name := range expectedTools {
		tool := getToolFromRegistry(name)
		if tool == nil {
			t.Errorf("Tool '%s' not found in registry", name)
			continue
		}
		if tool.Name() != name {
			t.Errorf("Tool.Name() = %v, want '%s'", tool.Name(), name)
		}
	}
}

// ===== Bash Tool Tests =====

func TestBashTool_Run(t *testing.T) {
	tool := getToolFromRegistry("bash")
	if tool == nil {
		t.Fatal("bash tool not found in registry")
	}

	args := map[string]string{"command": "echo 'test'"}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("bash.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("bash.Run() change should be nil")
	}
	if !strings.Contains(output, "test") {
		t.Errorf("bash.Run() output = %v, should contain 'test'", output)
	}
}

// ===== Read File Tool Tests =====

func TestReadFileTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("read_file")
	if tool == nil {
		t.Fatal("read_file tool not found in registry")
	}

	args := map[string]string{"path": testFile}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("read_file.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("read_file.Run() change should be nil")
	}
	if !strings.Contains(output, content) {
		t.Errorf("read_file.Run() output = %v, should contain '%s'", output, content)
	}
}

// ===== Write File Tool Tests =====

func TestWriteFileTool_Run_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	content := "new content"

	tool := getToolFromRegistry("write_file")
	if tool == nil {
		t.Fatal("write_file tool not found in registry")
	}

	args := map[string]string{"path": testFile, "content": content}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("write_file.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("write_file.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("write_file.Run() output = %v, should contain 'Successfully wrote'", output)
	}

	// ファイル内容確認
	gotContent, _ := os.ReadFile(testFile)
	if string(gotContent) != content {
		t.Errorf("write_file.Run() wrote %v, want %v", string(gotContent), content)
	}
}

func TestWriteFileTool_Run_ExistingFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	oldContent := "old content"
	newContent := "new content"

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("write_file")
	if tool == nil {
		t.Fatal("write_file tool not found in registry")
	}

	args := map[string]string{"path": testFile, "content": newContent}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("write_file.Run() error = %v", err)
	}
	if change == nil {
		t.Error("write_file.Run() change should not be nil for existing file")
	} else {
		if change.FilePath != testFile {
			t.Errorf("write_file.Run() change.FilePath = %v, want %v", change.FilePath, testFile)
		}
		if change.Tool != "write_file" {
			t.Errorf("write_file.Run() change.Tool = %v, want 'write_file'", change.Tool)
		}
		if change.BackupPath == "" {
			t.Error("write_file.Run() change.BackupPath should not be empty")
		}
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("write_file.Run() output = %v", output)
	}
}

// ===== Str Replace Tool Tests =====

func TestStrReplaceTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "replace.txt")
	content := "Hello World\nHello Universe"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("str_replace")
	if tool == nil {
		t.Fatal("str_replace tool not found in registry")
	}

	args := map[string]string{
		"path":    testFile,
		"old_str": "World",
		"new_str": "Earth",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("str_replace.Run() error = %v", err)
	}
	if change == nil {
		t.Error("str_replace.Run() change should not be nil")
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("str_replace.Run() output = %v", output)
	}

	// ファイル内容確認
	gotContent, _ := os.ReadFile(testFile)
	if !strings.Contains(string(gotContent), "Earth") {
		t.Error("str_replace.Run() did not replace text")
	}
	if strings.Contains(string(gotContent), "World") {
		t.Error("str_replace.Run() should not contain old text")
	}
}

// ===== List Dir Tool Tests =====

func TestListDirTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("list_dir")
	if tool == nil {
		t.Fatal("list_dir tool not found in registry")
	}

	args := map[string]string{"path": tmpDir}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("list_dir.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("list_dir.Run() change should be nil")
	}
	if !strings.Contains(output, "test.txt") {
		t.Errorf("list_dir.Run() output = %v, should contain 'test.txt'", output)
	}
}

// ===== Git Status Tool Tests =====

func TestGitStatusTool_Run(t *testing.T) {
	tool := getToolFromRegistry("git_status")
	if tool == nil {
		t.Fatal("git_status tool not found in registry")
	}

	args := map[string]string{}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("git_status.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("git_status.Run() change should be nil")
	}
	// 出力は空またはgit statusの結果
	_ = output
}

// ===== Git Diff Tool Tests =====

func TestGitDiffTool_Run(t *testing.T) {
	tool := getToolFromRegistry("git_diff")
	if tool == nil {
		t.Fatal("git_diff tool not found in registry")
	}

	args := map[string]string{"path": "."}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("git_diff.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("git_diff.Run() change should be nil")
	}
	// 出力は空またはgit diffの結果
	_ = output
}

// ===== Git Log Tool Tests =====

func TestGitLogTool_Run(t *testing.T) {
	tool := getToolFromRegistry("git_log")
	if tool == nil {
		t.Fatal("git_log tool not found in registry")
	}

	args := map[string]string{}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("git_log.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("git_log.Run() change should be nil")
	}
	// 出力は空またはgit logの結果
	_ = output
}

// ===== Search Code Tool Tests =====

func TestSearchCodeTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\nfunc main() {}"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("search_code")
	if tool == nil {
		t.Fatal("search_code tool not found in registry")
	}

	args := map[string]string{"pattern": "func main", "path": tmpDir}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("search_code.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("search_code.Run() change should be nil")
	}
	if !strings.Contains(output, "func main") {
		t.Errorf("search_code.Run() output = %v, should contain 'func main'", output)
	}
}

// ===== Search File Tool Tests =====

func TestSearchFileTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("search_file")
	if tool == nil {
		t.Fatal("search_file tool not found in registry")
	}

	args := map[string]string{"pattern": "*.go", "path": tmpDir}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("search_file.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("search_file.Run() change should be nil")
	}
	if !strings.Contains(output, "test.go") {
		t.Errorf("search_file.Run() output = %v, should contain 'test.go'", output)
	}
}

// ===== Append File Tool Tests =====

func TestAppendFileTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append.txt")
	content := "appended content"

	tool := getToolFromRegistry("append_file")
	if tool == nil {
		t.Fatal("append_file tool not found in registry")
	}

	args := map[string]string{"path": testFile, "content": content}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("append_file.Run() error = %v", err)
	}
	// 新規ファイルの場合はchangeはnil
	if change != nil {
		t.Errorf("append_file.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("append_file.Run() output = %v", output)
	}
}

// ===== Prepend File Tool Tests =====

func TestPrependFileTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "prepend.txt")
	content := "prepended content"

	tool := getToolFromRegistry("prepend_file")
	if tool == nil {
		t.Fatal("prepend_file tool not found in registry")
	}

	args := map[string]string{"path": testFile, "content": content}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("prepend_file.Run() error = %v", err)
	}
	// 新規ファイルの場合はchangeはnil
	if change != nil {
		t.Errorf("prepend_file.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("prepend_file.Run() output = %v", output)
	}
}

// ===== Create Dir Tool Tests =====

func TestCreateDirTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	tool := getToolFromRegistry("create_dir")
	if tool == nil {
		t.Fatal("create_dir tool not found in registry")
	}

	args := map[string]string{"path": newDir}
	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("create_dir.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("create_dir.Run() change should be nil")
	}
	if !strings.Contains(output, "Successfully created") {
		t.Errorf("create_dir.Run() output = %v", output)
	}

	// ディレクトリ存在確認
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("create_dir.Run() did not create directory")
	}
}

// ===== Insert After Tool Tests =====

func TestInsertAfterTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "insert.txt")
	content := "Line 1\nLine 2\nLine 3"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("insert_after")
	if tool == nil {
		t.Fatal("insert_after tool not found in registry")
	}

	args := map[string]string{
		"path":    testFile,
		"pattern": "Line 2",
		"content": "Inserted",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("insert_after.Run() error = %v", err)
	}
	if change == nil {
		t.Error("insert_after.Run() change should not be nil")
	}
	if !strings.Contains(output, "Inserted after") {
		t.Errorf("insert_after.Run() output = %v", output)
	}
}

// ===== Insert Before Tool Tests =====

func TestInsertBeforeTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "insert.txt")
	content := "Line 1\nLine 2\nLine 3"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := getToolFromRegistry("insert_before")
	if tool == nil {
		t.Fatal("insert_before tool not found in registry")
	}

	args := map[string]string{
		"path":    testFile,
		"pattern": "Line 2",
		"content": "Inserted",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("insert_before.Run() error = %v", err)
	}
	if change == nil {
		t.Error("insert_before.Run() change should not be nil")
	}
	if !strings.Contains(output, "Inserted before") {
		t.Errorf("insert_before.Run() output = %v", output)
	}
}

// ===== Copy File Tool Tests =====

func TestCopyFileTool_Run(t *testing.T) {
	setupTestMocks(t)

	// confirmをモック化
	oldConfirm := confirm
	confirm = func(message string) bool { return true }
	t.Cleanup(func() { confirm = oldConfirm })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")
	content := "source content"

	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	tool := getToolFromRegistry("copy_file")
	if tool == nil {
		t.Fatal("copy_file tool not found in registry")
	}

	args := map[string]string{
		"src":  srcFile,
		"dest": destFile,
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("copy_file.Run() error = %v", err)
	}
	// 新規コピーの場合はchangeはnil
	if change != nil {
		t.Errorf("copy_file.Run() change should be nil for new copy")
	}
	if !strings.Contains(output, "copied") {
		t.Errorf("copy_file.Run() output = %v", output)
	}

	// コピー先の内容確認
	gotContent, _ := os.ReadFile(destFile)
	if string(gotContent) != content {
		t.Errorf("copy_file.Run() copied content = %v, want %v", string(gotContent), content)
	}
}

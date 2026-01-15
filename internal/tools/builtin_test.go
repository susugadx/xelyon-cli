package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===== Bash Tool Tests =====

func TestBashTool_Name(t *testing.T) {
	tool := &BashTool{}
	if tool.Name() != "bash" {
		t.Errorf("BashTool.Name() = %v, want 'bash'", tool.Name())
	}
}

func TestBashTool_Run(t *testing.T) {
	tool := &BashTool{}
	args := map[string]string{"command": "echo 'test'"}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("BashTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("BashTool.Run() change should be nil")
	}
	if !strings.Contains(output, "test") {
		t.Errorf("BashTool.Run() output = %v, should contain 'test'", output)
	}
}

// ===== Read File Tool Tests =====

func TestReadFileTool_Name(t *testing.T) {
	tool := &ReadFileTool{}
	if tool.Name() != "read_file" {
		t.Errorf("ReadFileTool.Name() = %v, want 'read_file'", tool.Name())
	}
}

func TestReadFileTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &ReadFileTool{}
	args := map[string]string{"path": testFile}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("ReadFileTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("ReadFileTool.Run() change should be nil")
	}
	if !strings.Contains(output, content) {
		t.Errorf("ReadFileTool.Run() output = %v, should contain '%s'", output, content)
	}
}

// ===== Write File Tool Tests =====

func TestWriteFileTool_Name(t *testing.T) {
	tool := &WriteFileTool{}
	if tool.Name() != "write_file" {
		t.Errorf("WriteFileTool.Name() = %v, want 'write_file'", tool.Name())
	}
}

func TestWriteFileTool_Run_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	content := "new content"

	tool := &WriteFileTool{}
	args := map[string]string{"path": testFile, "content": content}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("WriteFileTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("WriteFileTool.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("WriteFileTool.Run() output = %v, should contain 'Successfully wrote'", output)
	}

	// ファイル内容確認
	gotContent, _ := os.ReadFile(testFile)
	if string(gotContent) != content {
		t.Errorf("WriteFileTool.Run() wrote %v, want %v", string(gotContent), content)
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

	tool := &WriteFileTool{}
	args := map[string]string{"path": testFile, "content": newContent}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("WriteFileTool.Run() error = %v", err)
	}
	if change == nil {
		t.Error("WriteFileTool.Run() change should not be nil for existing file")
	} else {
		if change.FilePath != testFile {
			t.Errorf("WriteFileTool.Run() change.FilePath = %v, want %v", change.FilePath, testFile)
		}
		if change.Tool != "write_file" {
			t.Errorf("WriteFileTool.Run() change.Tool = %v, want 'write_file'", change.Tool)
		}
		if change.BackupPath == "" {
			t.Error("WriteFileTool.Run() change.BackupPath should not be empty")
		}
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("WriteFileTool.Run() output = %v", output)
	}
}

// ===== Str Replace Tool Tests =====

func TestStrReplaceTool_Name(t *testing.T) {
	tool := &StrReplaceTool{}
	if tool.Name() != "str_replace" {
		t.Errorf("StrReplaceTool.Name() = %v, want 'str_replace'", tool.Name())
	}
}

func TestStrReplaceTool_Run(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "replace.txt")
	content := "Hello World\nHello Universe"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &StrReplaceTool{}
	args := map[string]string{
		"path":    testFile,
		"old_str": "World",
		"new_str": "Earth",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("StrReplaceTool.Run() error = %v", err)
	}
	if change == nil {
		t.Error("StrReplaceTool.Run() change should not be nil")
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("StrReplaceTool.Run() output = %v", output)
	}

	// ファイル内容確認
	gotContent, _ := os.ReadFile(testFile)
	if !strings.Contains(string(gotContent), "Earth") {
		t.Error("StrReplaceTool.Run() did not replace text")
	}
	if strings.Contains(string(gotContent), "World") {
		t.Error("StrReplaceTool.Run() should not contain old text")
	}
}

// ===== List Dir Tool Tests =====

func TestListDirTool_Name(t *testing.T) {
	tool := &ListDirTool{}
	if tool.Name() != "list_dir" {
		t.Errorf("ListDirTool.Name() = %v, want 'list_dir'", tool.Name())
	}
}

func TestListDirTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &ListDirTool{}
	args := map[string]string{"path": tmpDir}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("ListDirTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("ListDirTool.Run() change should be nil")
	}
	if !strings.Contains(output, "test.txt") {
		t.Errorf("ListDirTool.Run() output = %v, should contain 'test.txt'", output)
	}
}

// ===== Git Status Tool Tests =====

func TestGitStatusTool_Name(t *testing.T) {
	tool := &GitStatusTool{}
	if tool.Name() != "git_status" {
		t.Errorf("GitStatusTool.Name() = %v, want 'git_status'", tool.Name())
	}
}

func TestGitStatusTool_Run(t *testing.T) {
	tool := &GitStatusTool{}
	args := map[string]string{}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("GitStatusTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("GitStatusTool.Run() change should be nil")
	}
	// 出力は空またはgit statusの結果
	_ = output
}

// ===== Git Diff Tool Tests =====

func TestGitDiffTool_Name(t *testing.T) {
	tool := &GitDiffTool{}
	if tool.Name() != "git_diff" {
		t.Errorf("GitDiffTool.Name() = %v, want 'git_diff'", tool.Name())
	}
}

func TestGitDiffTool_Run(t *testing.T) {
	tool := &GitDiffTool{}
	args := map[string]string{"path": "."}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("GitDiffTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("GitDiffTool.Run() change should be nil")
	}
	// 出力は空またはgit diffの結果
	_ = output
}

// ===== Git Add Tool Tests =====

func TestGitAddTool_Name(t *testing.T) {
	tool := &GitAddTool{}
	if tool.Name() != "git_add" {
		t.Errorf("GitAddTool.Name() = %v, want 'git_add'", tool.Name())
	}
}

// ===== Git Commit Tool Tests =====

func TestGitCommitTool_Name(t *testing.T) {
	tool := &GitCommitTool{}
	if tool.Name() != "git_commit" {
		t.Errorf("GitCommitTool.Name() = %v, want 'git_commit'", tool.Name())
	}
}

// ===== Git Push Tool Tests =====

func TestGitPushTool_Name(t *testing.T) {
	tool := &GitPushTool{}
	if tool.Name() != "git_push" {
		t.Errorf("GitPushTool.Name() = %v, want 'git_push'", tool.Name())
	}
}

// ===== Git Log Tool Tests =====

func TestGitLogTool_Name(t *testing.T) {
	tool := &GitLogTool{}
	if tool.Name() != "git_log" {
		t.Errorf("GitLogTool.Name() = %v, want 'git_log'", tool.Name())
	}
}

func TestGitLogTool_Run(t *testing.T) {
	tool := &GitLogTool{}
	args := map[string]string{}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("GitLogTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("GitLogTool.Run() change should be nil")
	}
	// 出力は空またはgit logの結果
	_ = output
}

// ===== Search Code Tool Tests =====

func TestSearchCodeTool_Name(t *testing.T) {
	tool := &SearchCodeTool{}
	if tool.Name() != "search_code" {
		t.Errorf("SearchCodeTool.Name() = %v, want 'search_code'", tool.Name())
	}
}

func TestSearchCodeTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\nfunc main() {}"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &SearchCodeTool{}
	args := map[string]string{"pattern": "func main", "path": tmpDir}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("SearchCodeTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("SearchCodeTool.Run() change should be nil")
	}
	if !strings.Contains(output, "func main") {
		t.Errorf("SearchCodeTool.Run() output = %v, should contain 'func main'", output)
	}
}

// ===== Search File Tool Tests =====

func TestSearchFileTool_Name(t *testing.T) {
	tool := &SearchFileTool{}
	if tool.Name() != "search_file" {
		t.Errorf("SearchFileTool.Name() = %v, want 'search_file'", tool.Name())
	}
}

func TestSearchFileTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &SearchFileTool{}
	args := map[string]string{"pattern": "*.go", "path": tmpDir}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("SearchFileTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("SearchFileTool.Run() change should be nil")
	}
	if !strings.Contains(output, "test.go") {
		t.Errorf("SearchFileTool.Run() output = %v, should contain 'test.go'", output)
	}
}

// ===== Web Search Tool Tests =====

func TestWebSearchTool_Name(t *testing.T) {
	tool := &WebSearchTool{}
	if tool.Name() != "web_search" {
		t.Errorf("WebSearchTool.Name() = %v, want 'web_search'", tool.Name())
	}
}

// ===== Append File Tool Tests =====

func TestAppendFileTool_Name(t *testing.T) {
	tool := &AppendFileTool{}
	if tool.Name() != "append_file" {
		t.Errorf("AppendFileTool.Name() = %v, want 'append_file'", tool.Name())
	}
}

func TestAppendFileTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append.txt")
	content := "appended content"

	tool := &AppendFileTool{}
	args := map[string]string{"path": testFile, "content": content}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("AppendFileTool.Run() error = %v", err)
	}
	// 新規ファイルの場合はchangeはnil
	if change != nil {
		t.Errorf("AppendFileTool.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("AppendFileTool.Run() output = %v", output)
	}
}

// ===== Prepend File Tool Tests =====

func TestPrependFileTool_Name(t *testing.T) {
	tool := &PrependFileTool{}
	if tool.Name() != "prepend_file" {
		t.Errorf("PrependFileTool.Name() = %v, want 'prepend_file'", tool.Name())
	}
}

func TestPrependFileTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "prepend.txt")
	content := "prepended content"

	tool := &PrependFileTool{}
	args := map[string]string{"path": testFile, "content": content}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("PrependFileTool.Run() error = %v", err)
	}
	// 新規ファイルの場合はchangeはnil
	if change != nil {
		t.Errorf("PrependFileTool.Run() change should be nil for new file")
	}
	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("PrependFileTool.Run() output = %v", output)
	}
}

// ===== Create Dir Tool Tests =====

func TestCreateDirTool_Name(t *testing.T) {
	tool := &CreateDirTool{}
	if tool.Name() != "create_dir" {
		t.Errorf("CreateDirTool.Name() = %v, want 'create_dir'", tool.Name())
	}
}

func TestCreateDirTool_Run(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	tool := &CreateDirTool{}
	args := map[string]string{"path": newDir}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("CreateDirTool.Run() error = %v", err)
	}
	if change != nil {
		t.Errorf("CreateDirTool.Run() change should be nil")
	}
	if !strings.Contains(output, "Successfully created") {
		t.Errorf("CreateDirTool.Run() output = %v", output)
	}

	// ディレクトリ存在確認
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("CreateDirTool.Run() did not create directory")
	}
}

// ===== Run Test Tool Tests =====

func TestRunTestTool_Name(t *testing.T) {
	tool := &RunTestTool{}
	if tool.Name() != "run_test" {
		t.Errorf("RunTestTool.Name() = %v, want 'run_test'", tool.Name())
	}
}

// ===== Format Tool Tests =====

func TestFormatTool_Name(t *testing.T) {
	tool := &FormatTool{}
	if tool.Name() != "format" {
		t.Errorf("FormatTool.Name() = %v, want 'format'", tool.Name())
	}
}

// ===== Insert After Tool Tests =====

func TestInsertAfterTool_Name(t *testing.T) {
	tool := &InsertAfterTool{}
	if tool.Name() != "insert_after" {
		t.Errorf("InsertAfterTool.Name() = %v, want 'insert_after'", tool.Name())
	}
}

func TestInsertAfterTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "insert.txt")
	content := "Line 1\nLine 2\nLine 3"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &InsertAfterTool{}
	args := map[string]string{
		"path":    testFile,
		"pattern": "Line 2",
		"content": "Inserted",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("InsertAfterTool.Run() error = %v", err)
	}
	if change == nil {
		t.Error("InsertAfterTool.Run() change should not be nil")
	}
	if !strings.Contains(output, "Inserted after") {
		t.Errorf("InsertAfterTool.Run() output = %v", output)
	}
}

// ===== Insert Before Tool Tests =====

func TestInsertBeforeTool_Name(t *testing.T) {
	tool := &InsertBeforeTool{}
	if tool.Name() != "insert_before" {
		t.Errorf("InsertBeforeTool.Name() = %v, want 'insert_before'", tool.Name())
	}
}

func TestInsertBeforeTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "insert.txt")
	content := "Line 1\nLine 2\nLine 3"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &InsertBeforeTool{}
	args := map[string]string{
		"path":    testFile,
		"pattern": "Line 2",
		"content": "Inserted",
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("InsertBeforeTool.Run() error = %v", err)
	}
	if change == nil {
		t.Error("InsertBeforeTool.Run() change should not be nil")
	}
	if !strings.Contains(output, "Inserted before") {
		t.Errorf("InsertBeforeTool.Run() output = %v", output)
	}
}

// ===== Copy File Tool Tests =====

func TestCopyFileTool_Name(t *testing.T) {
	tool := &CopyFileTool{}
	if tool.Name() != "copy_file" {
		t.Errorf("CopyFileTool.Name() = %v, want 'copy_file'", tool.Name())
	}
}

func TestCopyFileTool_Run(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

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

	tool := &CopyFileTool{}
	args := map[string]string{
		"src":  srcFile,
		"dest": destFile,
	}

	output, change, err := tool.Run(args)

	if err != nil {
		t.Errorf("CopyFileTool.Run() error = %v", err)
	}
	// 新規コピーの場合はchangeはnil
	if change != nil {
		t.Errorf("CopyFileTool.Run() change should be nil for new copy")
	}
	if !strings.Contains(output, "copied") {
		t.Errorf("CopyFileTool.Run() output = %v", output)
	}

	// コピー先の内容確認
	gotContent, _ := os.ReadFile(destFile)
	if string(gotContent) != content {
		t.Errorf("CopyFileTool.Run() copied content = %v, want %v", string(gotContent), content)
	}
}

// ===== Git Branch Tool Tests =====

func TestGitBranchTool_Name(t *testing.T) {
	tool := &GitBranchTool{}
	if tool.Name() != "git_branch" {
		t.Errorf("GitBranchTool.Name() = %v, want 'git_branch'", tool.Name())
	}
}

// ===== Git Checkout Tool Tests =====

func TestGitCheckoutTool_Name(t *testing.T) {
	tool := &GitCheckoutTool{}
	if tool.Name() != "git_checkout" {
		t.Errorf("GitCheckoutTool.Name() = %v, want 'git_checkout'", tool.Name())
	}
}

// ===== Git Stash Tool Tests =====

func TestGitStashTool_Name(t *testing.T) {
	tool := &GitStashTool{}
	if tool.Name() != "git_stash" {
		t.Errorf("GitStashTool.Name() = %v, want 'git_stash'", tool.Name())
	}
}

// ===== Delete Lines Tool Tests =====

func TestDeleteLinesTool_Name(t *testing.T) {
	tool := &DeleteLinesTool{}
	if tool.Name() != "delete_lines" {
		t.Errorf("DeleteLinesTool.Name() = %v, want 'delete_lines'", tool.Name())
	}
}

// ===== Delete File Tool Tests =====

func TestDeleteFileTool_Name(t *testing.T) {
	tool := &DeleteFileTool{}
	if tool.Name() != "delete_file" {
		t.Errorf("DeleteFileTool.Name() = %v, want 'delete_file'", tool.Name())
	}
}

// ===== Move File Tool Tests =====

func TestMoveFileTool_Name(t *testing.T) {
	tool := &MoveFileTool{}
	if tool.Name() != "move_file" {
		t.Errorf("MoveFileTool.Name() = %v, want 'move_file'", tool.Name())
	}
}

// ===== Lint Tool Tests =====

func TestLintTool_Name(t *testing.T) {
	tool := &LintTool{}
	if tool.Name() != "lint" {
		t.Errorf("LintTool.Name() = %v, want 'lint'", tool.Name())
	}
}

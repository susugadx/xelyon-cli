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
	// 19ツール（16ツールはbash/str_replaceで代用可能なため削除済み）
	expectedTools := []string{
		// File Operations (7)
		"read_file", "write_file", "str_replace", "delete_file",
		"list_dir", "restore_backup", "list_backups",
		// Git Operations (2)
		"git_commit", "git_checkout",
		// Search Operations (5)
		"search_code", "search_file", "web_search", "ast_grep", "grep_replace",
		// Development Operations (5)
		"bash", "run_test", "format", "lint", "http_request",
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

// ===== parseInt Tests =====

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "positive integer",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   "-10",
			want:    -10,
			wantErr: false,
		},
		{
			name:    "large positive",
			input:   "999999",
			want:    999999,
			wantErr: false,
		},
		{
			name:    "large negative",
			input:   "-123456",
			want:    -123456,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "non-numeric string",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "float string",
			input:   "3.14",
			want:    0,
			wantErr: true,
		},
		{
			name:    "string with spaces",
			input:   " 10",
			want:    0,
			wantErr: true,
		},
		{
			name:    "mixed alphanumeric",
			input:   "123abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "plus sign prefix",
			input:   "+5",
			want:    5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

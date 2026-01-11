package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteCopyFile_Success(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")
	content := "Test content\nLine 2"

	// ソースファイル作成
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	output, backupPath, err := executeCopyFile(srcFile, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	// バックアップは作成されない（新規コピー）
	if backupPath != "" {
		t.Errorf("executeCopyFile() backupPath = %v, want empty for new copy", backupPath)
	}

	// コピー先ファイルが作成されたことを確認
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("executeCopyFile() did not create destination file")
	}

	// 内容を確認
	gotContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(gotContent) != content {
		t.Errorf("executeCopyFile() file content = %v, want %v", string(gotContent), content)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "File copied") {
		t.Errorf("executeCopyFile() output = %v, should contain 'File copied'", output)
	}
}

func TestExecuteCopyFile_Overwrite(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	// confirmをモック化（自動承認）
	oldConfirm := confirm
	confirm = func(message string) bool {
		return true
	}
	t.Cleanup(func() { confirm = oldConfirm })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")
	srcContent := "New content"
	destContent := "Old content"

	// ソースとコピー先ファイル作成
	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(destFile, []byte(destContent), 0644); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	output, backupPath, err := executeCopyFile(srcFile, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	// バックアップが作成されたことを確認
	if backupPath == "" {
		t.Error("executeCopyFile() should create backup when overwriting")
	}

	// コピー先ファイルの内容が更新されたことを確認
	gotContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(gotContent) != srcContent {
		t.Errorf("executeCopyFile() file content = %v, want %v", string(gotContent), srcContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "overwritten") {
		t.Errorf("executeCopyFile() output = %v, should contain 'overwritten'", output)
	}
}

func TestExecuteCopyFile_SourceNotFound(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "notexist.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")

	output, backupPath, err := executeCopyFile(srcFile, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeCopyFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "not found") {
		t.Errorf("executeCopyFile() output = %v, should contain error about file not found", output)
	}
}

func TestExecuteCopyFile_SourceIsDirectory(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source_dir")
	destFile := filepath.Join(tmpDir, "dest.txt")

	// ソースディレクトリ作成
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	output, backupPath, err := executeCopyFile(srcDir, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeCopyFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "directory") {
		t.Errorf("executeCopyFile() output = %v, should contain error about directory", output)
	}
}

func TestExecuteCopyFile_PreservePermissions(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")
	content := "Test content"

	// ソースファイル作成（0600パーミッション）
	if err := os.WriteFile(srcFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	_, _, err := executeCopyFile(srcFile, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	// パーミッション確認
	srcInfo, err := os.Stat(srcFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	destInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if srcInfo.Mode() != destInfo.Mode() {
		t.Errorf("executeCopyFile() did not preserve permissions: src=%v, dest=%v", srcInfo.Mode(), destInfo.Mode())
	}
}

func TestExecuteCopyFile_CancelledByUser(t *testing.T) {
	// ValidatePathをモック化
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })

	// confirmをモック化（拒否）
	oldConfirm := confirm
	confirm = func(message string) bool {
		return false
	}
	t.Cleanup(func() { confirm = oldConfirm })

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "dest.txt")

	// ソースとコピー先ファイル作成
	if err := os.WriteFile(srcFile, []byte("src"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.WriteFile(destFile, []byte("dest"), 0644); err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	output, backupPath, err := executeCopyFile(srcFile, destFile)
	if err != nil {
		t.Fatalf("executeCopyFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeCopyFile() backupPath = %v, want empty when cancelled", backupPath)
	}

	if !strings.Contains(output, "Cancelled") {
		t.Errorf("executeCopyFile() output = %v, should contain 'Cancelled'", output)
	}

	// コピー先ファイルが変更されていないことを確認
	gotContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(gotContent) != "dest" {
		t.Error("executeCopyFile() should not modify destination when cancelled")
	}
}

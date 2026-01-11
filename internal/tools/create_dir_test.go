package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteCreateDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	output := executeCreateDir(newDir)

	// ディレクトリが作成されたことを確認
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("executeCreateDir() did not create directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("executeCreateDir() created path is not a directory")
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Successfully created") {
		t.Errorf("executeCreateDir() output = %v, should contain 'Successfully created'", output)
	}

	// パーミッション確認（0755）
	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0755)
	if mode != expectedMode {
		t.Errorf("executeCreateDir() directory mode = %v, want %v", mode, expectedMode)
	}
}

func TestExecuteCreateDir_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "level3")

	output := executeCreateDir(nestedDir)

	// ネストされたディレクトリが作成されたことを確認
	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("executeCreateDir() did not create nested directories: %v", err)
	}

	if !info.IsDir() {
		t.Error("executeCreateDir() created path is not a directory")
	}

	if !strings.Contains(output, "Successfully created") {
		t.Errorf("executeCreateDir() output = %v, should contain 'Successfully created'", output)
	}
}

func TestExecuteCreateDir_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "existing")

	// ディレクトリを事前に作成
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	output := executeCreateDir(existingDir)

	// 出力メッセージを確認（べき等性）
	if !strings.Contains(output, "already exists") {
		t.Errorf("executeCreateDir() output = %v, should contain 'already exists'", output)
	}

	// ディレクトリが存在することを確認
	info, err := os.Stat(existingDir)
	if err != nil {
		t.Fatalf("executeCreateDir() removed existing directory: %v", err)
	}

	if !info.IsDir() {
		t.Error("executeCreateDir() path is not a directory")
	}
}

func TestExecuteCreateDir_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "file.txt")

	// ファイルを作成
	if err := os.WriteFile(existingFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := executeCreateDir(existingFile)

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "not a directory") {
		t.Errorf("executeCreateDir() output = %v, should contain error about path not being a directory", output)
	}
}

func TestExecuteCreateDir_EmptyPath(t *testing.T) {
	output := executeCreateDir("")

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "empty") {
		t.Errorf("executeCreateDir() output = %v, should contain error about empty path", output)
	}
}

func TestExecuteCreateDir_MultipleCreations(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	dir3 := filepath.Join(tmpDir, "dir3")

	// 複数のディレクトリを作成
	output1 := executeCreateDir(dir1)
	output2 := executeCreateDir(dir2)
	output3 := executeCreateDir(dir3)

	// すべて成功していることを確認
	if !strings.Contains(output1, "Successfully created") {
		t.Errorf("executeCreateDir() output1 = %v", output1)
	}
	if !strings.Contains(output2, "Successfully created") {
		t.Errorf("executeCreateDir() output2 = %v", output2)
	}
	if !strings.Contains(output3, "Successfully created") {
		t.Errorf("executeCreateDir() output3 = %v", output3)
	}

	// すべてのディレクトリが存在することを確認
	for _, dir := range []string{dir1, dir2, dir3} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("executeCreateDir() did not create directory: %s", dir)
		}
	}
}

func TestExecuteCreateDir_WithSpacesInName(t *testing.T) {
	tmpDir := t.TempDir()
	dirWithSpaces := filepath.Join(tmpDir, "dir with spaces")

	output := executeCreateDir(dirWithSpaces)

	// ディレクトリが作成されたことを確認
	info, err := os.Stat(dirWithSpaces)
	if err != nil {
		t.Fatalf("executeCreateDir() did not create directory with spaces: %v", err)
	}

	if !info.IsDir() {
		t.Error("executeCreateDir() created path is not a directory")
	}

	if !strings.Contains(output, "Successfully created") {
		t.Errorf("executeCreateDir() output = %v", output)
	}
}

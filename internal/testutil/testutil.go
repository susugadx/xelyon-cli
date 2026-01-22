package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// MockConfirm はconfirm()関数をモックするヘルパー
func MockConfirm(result bool) func(string) bool {
	return func(prompt string) bool {
		return result
	}
}

// CreateTempFile は一時ファイルを作成するヘルパー
func CreateTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)

	// ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return path
}

// AssertFileExists はファイルが存在することを確認
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File does not exist: %s", path)
	}
}

// AssertFileNotExists はファイルが存在しないことを確認
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("File should not exist: %s", path)
	}
}

// AssertFileContent はファイル内容が期待通りであることを確認
func AssertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if string(content) != expected {
		t.Errorf("File content mismatch for %s:\nGot:\n%s\nExpected:\n%s", path, string(content), expected)
	}
}

// SetupTempHome は一時的なHOMEディレクトリを設定するヘルパー
func SetupTempHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")

	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	})

	return tmpDir
}

// ReadFile はファイルを読み込むヘルパー
// エラー時は t.Fatalf() でテストを失敗させる
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// FileExists はファイルが存在するかチェック（boolを返す）
func FileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

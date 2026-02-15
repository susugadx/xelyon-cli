package agent

import (
	"os"
	"testing"
)

func TestFileExists(t *testing.T) {
	// 存在するファイル
	tmpFile, err := os.CreateTemp("", "xelyon-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !fileExists(tmpFile.Name()) {
		t.Error("fileExists() = false for existing file")
	}

	// 存在しないファイル
	if fileExists("/nonexistent/file/path") {
		t.Error("fileExists() = true for nonexistent file")
	}
}

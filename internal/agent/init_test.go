package agent

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateXELYONMDTemplate(t *testing.T) {
	projectName := "test-project"
	result := generateXELYONMDTemplate(projectName)

	// プロジェクト名が含まれているか
	if !strings.Contains(result, "# test-project") {
		t.Errorf("generateXELYONMDTemplate() missing project name")
	}

	// AI用コンテキストの注意書きが含まれているか
	if !strings.Contains(result, "AI 用コンテキスト") {
		t.Errorf("generateXELYONMDTemplate() missing AI context warning")
	}

	// 肥大化禁止の警告が含まれているか
	if !strings.Contains(result, "肥大化させることを禁止") {
		t.Errorf("generateXELYONMDTemplate() missing anti-bloat warning")
	}

	// 概要セクションが含まれているか
	if !strings.Contains(result, "## 概要") {
		t.Errorf("generateXELYONMDTemplate() missing overview section")
	}

	// 開発ルールセクションが含まれているか
	if !strings.Contains(result, "## 開発ルール") {
		t.Errorf("generateXELYONMDTemplate() missing rules section")
	}

	// Verification Commandsセクションが含まれているか
	if !strings.Contains(result, "## Verification Commands") {
		t.Errorf("generateXELYONMDTemplate() missing Verification Commands section")
	}

	// テンプレートが25行以下であることを確認
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) > 25 {
		t.Errorf("generateXELYONMDTemplate() generated %d lines, expected <= 25", len(lines))
	}
}

func TestGenerateXELYONMDTemplate_DifferentProjectNames(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
	}{
		{"simple name", "myproject"},
		{"name with hyphen", "my-project"},
		{"name with underscore", "my_project"},
		{"name with numbers", "project123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateXELYONMDTemplate(tt.projectName)
			if !strings.Contains(result, "# "+tt.projectName) {
				t.Errorf("generateXELYONMDTemplate(%s) missing project name in title", tt.projectName)
			}
		})
	}
}

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

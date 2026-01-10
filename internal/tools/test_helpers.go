package tools

import (
	"path/filepath"
	"testing"
)

// setupTestMocks はテスト用の共通モックを設定
func setupTestMocks(t *testing.T) {
	// ValidatePathをモック（テスト時はパストラバーサルチェックをスキップ）
	originalValidate := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() {
		ValidatePath = originalValidate
	})
}

// setupTestConfirm はconfirm関数をモック
func setupTestConfirm(t *testing.T, result bool) {
	originalConfirm := confirm
	confirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		confirm = originalConfirm
	})
}

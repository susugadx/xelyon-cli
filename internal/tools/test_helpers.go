package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestMocks はテスト用の共通モックを設定
func setupTestMocks(t *testing.T) {
	// 対話モードを無効化（シンプルなy/n確認を使う）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	// confirm関数をモック（デフォルトは自動承認）
	setupTestConfirm(t, true)

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

// setupTestConfirmInteractive はConfirmInteractive関数をモック
// action: "yes", "no", "comment"
func setupTestConfirmInteractive(t *testing.T, action string) {
	originalConfirmInteractive := ConfirmInteractive
	ConfirmInteractive = func(msg string) ConfirmResult {
		return ConfirmResult{Action: action}
	}
	t.Cleanup(func() {
		ConfirmInteractive = originalConfirmInteractive
	})
}

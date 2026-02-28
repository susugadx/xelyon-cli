package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// setupTestMocks はテスト用の共通モックを設定
func setupTestMocks(t *testing.T) {
	// 対話モードを無効化（シンプルなy/n確認を使う）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	// テスト中に .gitignore 追記のプロンプトを出さない
	os.Setenv("XELYON_SKIP_GITIGNORE_PROMPT", "1")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		os.Unsetenv("XELYON_SKIP_GITIGNORE_PROMPT")
	})

	// confirm関数をモック（デフォルトは自動承認）
	setupTestConfirm(t, true)

	// ValidatePathをモック（テスト時はパストラバーサルチェックをスキップ）
	originalValidate := common.ValidatePath
	common.ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() {
		common.ValidatePath = originalValidate
	})
}

// setupTestConfirm はconfirm関数をモック
func setupTestConfirm(t *testing.T, result bool) {
	originalConfirm := common.SimpleConfirm
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
	})
}

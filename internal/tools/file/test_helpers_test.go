package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
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
	originalConfirmWithIO := common.SimpleConfirmWithIO
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	common.SimpleConfirmWithIO = func(_ ui.PromptIO, msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
		common.SimpleConfirmWithIO = originalConfirmWithIO
	})
}

func testConfirmOptions() common.ConfirmOptions {
	return common.ConfirmOptions{Config: config.DefaultConfig()}
}

func testPromptIO(stdout, stderr io.Writer) ui.PromptIO {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return ui.NewPromptIO(strings.NewReader(""), stdout, stderr, nil)
}

func executeWriteFileForTest(path, content string) (string, error) {
	return ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(testPromptIO(nil, nil), testConfirmOptions(), nil, path, content)
}

func executeDeleteFileForTest(path string) (string, error) {
	return ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(testPromptIO(nil, nil), testConfirmOptions(), nil, path)
}

func executeStrReplaceForTest(path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	return ExecuteStrReplaceWithPromptIOAndOptions(testPromptIO(nil, nil), testConfirmOptions(), path, oldStr, newStr, startLineStr, endLineStr)
}

func executeStrReplaceWithWritersForTest(stdout, stderr io.Writer, path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	return ExecuteStrReplaceWithPromptIOAndOptions(testPromptIO(stdout, stderr), testConfirmOptions(), path, oldStr, newStr, startLineStr, endLineStr)
}

func executeBatchEditsForTest(path, editsJSON string) (string, error) {
	return executeBatchEditsWithPromptIOAndOptions(testPromptIO(nil, nil), testConfirmOptions(), path, editsJSON)
}

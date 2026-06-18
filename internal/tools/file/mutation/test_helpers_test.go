package mutation

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// setupTestEnvironment は file tool テスト共通の環境変数を設定する。
func setupTestEnvironment(t *testing.T) {
	// 対話モードを無効化（シンプルなy/n確認を使う）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	// テスト中に .gitignore 追記のプロンプトを出さない
	os.Setenv("XELYON_SKIP_GITIGNORE_PROMPT", "1")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		os.Unsetenv("XELYON_SKIP_GITIGNORE_PROMPT")
	})
}

// withPermissiveValidatePath は TempDir を使うテスト向けに ValidatePath を一時的に緩和する。
func withPermissiveValidatePath(t *testing.T) func() {
	t.Helper()
	originalValidate := common.ValidatePath
	common.ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	return func() {
		common.ValidatePath = originalValidate
	}
}

// setupTestMocks は file tool テストの既定モックをまとめて設定する。
func setupTestMocks(t *testing.T) {
	setupTestEnvironment(t)
	setupTestConfirm(t, true)
}

// setupTestConfirm は confirm 関数をモックする。
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

func newTestToolExecContext() tools.ExecutionContext {
	return tools.ExecutionContext{
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
}

func assertNoFileChange(t *testing.T, change *tools.FileChange) {
	t.Helper()
	if change != nil {
		t.Fatalf("expected nil change, got: %+v", change)
	}
}

func assertHasFileChange(t *testing.T, change *tools.FileChange) {
	t.Helper()
	if change == nil {
		t.Fatal("expected non-nil change")
	}
}

func assertFileChangeResolvedPath(t *testing.T, change *tools.FileChange, want string) {
	t.Helper()
	if change == nil {
		t.Fatal("expected non-nil change")
	}
	if len(change.Details) != 1 {
		t.Fatalf("len(change.Details) = %d, want 1", len(change.Details))
	}
	got := filepath.Clean(change.Details[0].FilePath)
	if got != filepath.Clean(want) {
		t.Fatalf("change.Details[0].FilePath = %q, want %q", got, filepath.Clean(want))
	}
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

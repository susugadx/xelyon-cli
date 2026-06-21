package readtool

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func setupTestEnvironment(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	os.Setenv("XELYON_SKIP_GITIGNORE_PROMPT", "1")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		os.Unsetenv("XELYON_SKIP_GITIGNORE_PROMPT")
	})
}

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

func setupTestMocks(t *testing.T) {
	setupTestEnvironment(t)
	setupTestConfirm(t, true)
}

func setupTestConfirm(t *testing.T, result bool) {
	originalConfirm := common.SimpleConfirm
	originalConfirmWithIO := common.SimpleConfirmWithIO
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	common.SimpleConfirmWithIO = func(_ uiruntime.PromptIO, msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
		common.SimpleConfirmWithIO = originalConfirmWithIO
	})
}

func newTestToolExecContext() tools.ExecutionContext {
	return tools.ExecutionContext{
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
}

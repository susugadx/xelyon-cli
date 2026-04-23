package dev

import (
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// setupTestMocks sets up test mocks (auto-approve)
func setupTestMocks(t *testing.T) {
	t.Helper()
	// Disable interactive mode for tests
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	originalConfirm := common.SimpleConfirm
	originalConfirmWithIO := common.SimpleConfirmWithIO
	common.SimpleConfirm = func(message string) bool {
		return true
	}
	common.SimpleConfirmWithIO = func(_ ui.PromptIO, message string) bool {
		return true
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
		common.SimpleConfirmWithIO = originalConfirmWithIO
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})
}

// setupTestConfirm sets up test confirm with specified result
func setupTestConfirm(t *testing.T, approve bool) {
	t.Helper()
	// Disable interactive mode for tests
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	originalConfirm := common.SimpleConfirm
	originalConfirmWithIO := common.SimpleConfirmWithIO
	common.SimpleConfirm = func(message string) bool {
		return approve
	}
	common.SimpleConfirmWithIO = func(_ ui.PromptIO, message string) bool {
		return approve
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
		common.SimpleConfirmWithIO = originalConfirmWithIO
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})
}

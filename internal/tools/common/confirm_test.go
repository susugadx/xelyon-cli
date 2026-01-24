package common

import (
	"os"
	"testing"
)

// setupTestConfirm はSimpleConfirmをモックする
func setupTestConfirm(t *testing.T, result bool) {
	t.Helper()
	original := SimpleConfirm
	SimpleConfirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		SimpleConfirm = original
	})
}

func TestConfirm_InteractiveDisabled_Yes(t *testing.T) {
	// インタラクティブモードを無効化
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, true)

	dec := Confirm("Proceed?")
	if dec.Action != ConfirmYes {
		t.Errorf("expected ConfirmYes, got %q", dec.Action)
	}
}

func TestConfirm_InteractiveDisabled_No(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, false)

	dec := Confirm("Proceed?")
	if dec.Action != ConfirmNo {
		t.Errorf("expected ConfirmNo, got %q", dec.Action)
	}
}

func TestConfirmApproved_Yes(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, true)

	if !ConfirmApproved("Proceed?") {
		t.Error("expected true")
	}
}

func TestConfirmApproved_No(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, false)

	if ConfirmApproved("Proceed?") {
		t.Error("expected false")
	}
}

func TestConfirmWithFeedback_Yes(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, true)

	approved, comment, image := ConfirmWithFeedback("Proceed?")
	if !approved {
		t.Error("expected approved to be true")
	}
	if comment != "" {
		t.Errorf("expected empty comment, got %q", comment)
	}
	if image != nil {
		t.Error("expected image to be nil")
	}
}

func TestConfirmWithFeedback_No(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	setupTestConfirm(t, false)

	approved, comment, image := ConfirmWithFeedback("Proceed?")
	if approved {
		t.Error("expected approved to be false")
	}
	if comment != "" {
		t.Errorf("expected empty comment, got %q", comment)
	}
	if image != nil {
		t.Error("expected image to be nil")
	}
}

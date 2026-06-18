package common

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// setupTestConfirm はSimpleConfirmをモックする
func setupTestConfirm(t *testing.T, result bool) {
	t.Helper()
	original := SimpleConfirm
	originalWithIO := SimpleConfirmWithIO
	SimpleConfirm = func(msg string) bool {
		return result
	}
	SimpleConfirmWithIO = func(_ uiruntime.PromptIO, msg string) bool {
		return result
	}
	t.Cleanup(func() {
		SimpleConfirm = original
		SimpleConfirmWithIO = originalWithIO
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

func TestConfirmWithIO_UsesInjectedIO(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	var out bytes.Buffer
	dec := ConfirmWithIO(uiruntime.NewPromptIO(strings.NewReader("y\n"), &out, io.Discard, nil), "Proceed?")
	if dec.Action != ConfirmYes {
		t.Fatalf("expected ConfirmYes, got %q", dec.Action)
	}
	if !strings.Contains(out.String(), "Proceed? (y/n): ") {
		t.Fatalf("expected prompt in injected stdout, got %q", out.String())
	}
}

func TestConfirmWithIO_DiscardDoesNotLeakProcessStdout(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	dec := ConfirmWithIO(uiruntime.NewPromptIO(strings.NewReader("y\n"), io.Discard, io.Discard, nil), "Proceed?")
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if dec.Action != ConfirmYes {
		t.Fatalf("expected ConfirmYes, got %q", dec.Action)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no process stdout leak, got %q", buf.String())
	}
}

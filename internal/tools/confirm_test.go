package tools

import "testing"

func TestConfirm_InteractiveDisabled_UsesLegacyConfirm(t *testing.T) {
	setupTestConfirm(t, true)

	// interactive モードOFF相当（envに依存するが、ConfirmはOFF時に confirm() を使う）
	// このテストは、confirm() が true を返した場合に ConfirmYes になることを確認する。
	dec := Confirm("Proceed?")
	if dec.Action != ConfirmYes {
		t.Fatalf("expected ConfirmYes, got %q", dec.Action)
	}
}

func TestConfirmApproved_InteractiveDisabled(t *testing.T) {
	setupTestConfirm(t, false)
	if ConfirmApproved("Proceed?") {
		t.Fatalf("expected false")
	}
}

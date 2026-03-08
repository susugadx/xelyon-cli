package common

import "testing"

func TestPushQuietMode_Nested(t *testing.T) {
	if IsQuietMode() {
		t.Fatal("quiet mode should start disabled")
	}

	restoreOuter := PushQuietMode()
	if !IsQuietMode() {
		t.Fatal("quiet mode should be enabled after outer push")
	}

	restoreInner := PushQuietMode()
	if !IsQuietMode() {
		t.Fatal("quiet mode should remain enabled after nested push")
	}

	restoreOuter()
	if !IsQuietMode() {
		t.Fatal("quiet mode should remain enabled while nested push is active")
	}

	restoreOuter()
	if !IsQuietMode() {
		t.Fatal("quiet mode should remain enabled after duplicate restore while nested push is active")
	}

	restoreInner()
	if IsQuietMode() {
		t.Fatal("quiet mode should be disabled after all restores run")
	}

	restoreInner()
	if IsQuietMode() {
		t.Fatal("quiet mode should remain disabled after duplicate nested restore")
	}
}

package tui

import "testing"

func TestActivateModalScreen_ExitsNavigationState(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	m.navigationMode = true
	m.gPressed = true
	m.pendingCount = 12
	m.yPressed = true
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: 3, col: 0}
	m.focusedBlock = 1

	m.activateModalScreen(screenConfig)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if m.navigationMode {
		t.Fatal("activateModalScreen should exit navigation mode")
	}
	if m.visualMode != visualModeOff || m.visualStart.line != -1 || m.visualStart.col != -1 {
		t.Fatalf("visual selection = mode %d start %#v, want cleared", m.visualMode, m.visualStart)
	}
	if m.gPressed || m.pendingCount != 0 || m.yPressed {
		t.Fatalf("nav pending state = g:%v count:%d y:%v, want reset", m.gPressed, m.pendingCount, m.yPressed)
	}
	if m.focusedBlock != -1 {
		t.Fatalf("focusedBlock = %d, want -1", m.focusedBlock)
	}
	if !m.chromeDirty {
		t.Fatal("chromeDirty = false, want true")
	}
}

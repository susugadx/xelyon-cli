package tui

import "time"

// setTransientStatus は一時通知メッセージを設定する。
func (m *Model) setTransientStatus(text string) {
	m.transientStatus = text
	m.transientStatusUntil = time.Now().Add(2 * time.Second)
	m.chromeDirty = true
}

func (m *Model) resetNavPending() {
	m.gPressed = false
	m.pendingCount = 0
	m.yPressed = false
}

func (m *Model) afterViewportScroll() {
	if m.navigationMode && m.focusedBlock < 0 && m.visualMode == visualModeOff {
		m.clampCursorToViewport()
	}
	if m.vp.atBottom() {
		m.newOutput = false
	}
	m.chromeDirty = true
}

package tui

import "time"

type navigationState struct {
	navigationMode bool           // Vim ナビゲーションモード
	gPressed       bool           // g キーが1回押された状態
	pendingCount   int            // 数字プレフィックスで入力中の移動回数
	yPressed       bool           // y キーが1回押された状態（yy用）
	cursorLine     int            // NAVモードの現在行（rawLines基準）
	cursorCol      int            // NAVモードの現在列（表示幅基準）
	visualMode     int            // 0=OFF, 1=文字単位, 2=行単位
	visualStart    visualPosition // visual selection の開始位置
}

type streamState struct {
	streamingActive   bool
	streamCursorCol   int
	streamActiveANSI  string
	streamPendingANSI string
}

type mouseSelectionState struct {
	mouseSelAnchor     visualPosition // マウスドラッグ選択の開始位置（line=-1=選択なし）
	mouseSelEnd        visualPosition // マウスドラッグ選択の終了位置
	mouseDragging      bool           // マウスドラッグ中か
	mouseAutoScrolling bool           // 自動スクロールティック発行済みか
	mouseLastScreenX   int            // 最後のマウスX座標（自動スクロール用）
	mouseLastScreenY   int            // 最後のマウスY座標（自動スクロール用）
}

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

// exitNavigationMode は navigation mode 固有の状態を破棄して通常状態に戻す。
func (m *Model) exitNavigationMode() bool {
	if !m.navigationMode {
		return false
	}
	m.clearVisualSelection()
	if m.focusedBlock >= 0 {
		m.clearBlockFocus()
	}
	m.navigationMode = false
	m.resetNavPending()
	m.chromeDirty = true
	return true
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

package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func normalizeRawLine(line string) string {
	line = strings.TrimSuffix(line, "\r")
	// VS16 (U+FE0F) を除去して emoji presentation による幅ズレを防ぐ。
	if strings.ContainsRune(line, '\uFE0F') {
		line = strings.ReplaceAll(line, "\uFE0F", "")
	}
	return line
}

// appendMessage は会話ログにメッセージを追加する。
func (m *Model) appendMessage(msg ChatMessage) tea.Cmd {
	m.messages = append(m.messages, msg)

	switch msg.Role {
	case "user":
		lines := strings.Split(msg.Content, "\n")
		rendered := make([]string, 0, len(lines)+2)
		rendered = append(rendered, "")
		for _, line := range lines {
			rendered = append(rendered, "> "+line)
		}
		rendered = append(rendered, "")
		return m.appendContentLines(rendered...)
	default:
		return m.appendContentLines(strings.Split(msg.Content, "\n")...)
	}
}

// appendSystemInfo はシステム情報メッセージを追加する。
func (m *Model) appendSystemInfo(text string) tea.Cmd {
	return m.appendMessage(ChatMessage{
		Role:      "system_info",
		Content:   text,
		Timestamp: time.Now(),
	})
}

// syncViewportContent は viewport の内容を更新し、auto-follow を制御する。
// 最下部にいる場合のみ追従し、上スクロール中は位置を維持する。
func (m *Model) syncViewportContent() {
	if !m.ready {
		return
	}
	atBottom := m.vp.atBottom()
	m.vp.setLines(m.getVisualRowContents())
	if atBottom {
		m.vp.gotoBottom()
		m.newOutput = false
	} else {
		m.newOutput = true
	}
}

// appendContentLines は生ログと描画済みログの両方に新しい行を追加する。
func (m *Model) appendContentLines(lines ...string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = normalizeRawLine(line)
	}
	m.rawLines = append(m.rawLines, normalized...)
	m.rebuildLayout()
	m.clampCursorLine()
	m.syncViewportContent()
	return nil
}

func (m *Model) clampCursorToViewport() {
	if len(m.rawLines) == 0 || m.vp.height <= 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	top := 0
	if m.layout != nil && m.vp.yOffset < len(m.layout.Rows) {
		top = m.layout.Rows[m.vp.yOffset].RawLineIdx
	}
	bottomIdx := m.vp.yOffset + m.vp.height - 1
	if m.layout != nil && bottomIdx >= len(m.layout.Rows) {
		bottomIdx = len(m.layout.Rows) - 1
	}
	bottom := len(m.rawLines) - 1
	if m.layout != nil && bottomIdx >= 0 && bottomIdx < len(m.layout.Rows) {
		bottom = m.layout.Rows[bottomIdx].RawLineIdx
	}
	if m.cursorLine < top {
		m.cursorLine = top
	}
	if m.cursorLine > bottom {
		m.cursorLine = bottom
	}
	m.clampCursorCol()
}

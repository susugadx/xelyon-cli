package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// appendMessage は会話ログにメッセージを追加する。
func (m *Model) appendMessage(msg ChatMessage) tea.Cmd {
	m.messages = append(m.messages, msg)

	switch msg.Role {
	case "user":
		return m.appendContentLines("", "> "+msg.Content, "")
	default:
		return m.appendContentLines(strings.Split(msg.Content, "\n")...)
	}
}

// appendStreamText はストリーミングテキストを追加する。
func (m *Model) appendStreamText(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, "\n")

	if !m.streamingActive {
		m.streamingActive = true
		return m.appendContentLines(parts...)
	}

	if len(m.rawLines) == 0 {
		return m.appendContentLines(parts...)
	}

	last := len(m.rawLines) - 1
	m.rawLines[last] += parts[0]
	if len(m.renderedLines) > last {
		m.renderedLines[last] = m.renderLine(m.rawLines[last])
	}
	if len(parts) > 1 {
		_ = m.appendContentLines(parts[1:]...)
	}
	m.clampCursorLine()
	if m.ready {
		atBottom := m.vp.atBottom()
		m.vp.setLines(m.renderedLines)
		if atBottom {
			m.vp.gotoBottom()
			m.newOutput = false
		} else {
			m.newOutput = true
		}
	}
	return nil
}

// appendSystemInfo はシステム情報メッセージを追加する。
func (m *Model) appendSystemInfo(text string) tea.Cmd {
	return m.appendMessage(ChatMessage{
		Role:      "system_info",
		Content:   text,
		Timestamp: time.Now(),
	})
}

// appendContentLines は生ログと描画済みログの両方に新しい行を追加する。
func (m *Model) appendContentLines(lines ...string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	m.rawLines = append(m.rawLines, lines...)
	for _, line := range lines {
		m.renderedLines = append(m.renderedLines, m.renderLine(line))
	}
	m.clampCursorLine()
	if m.ready {
		atBottom := m.vp.atBottom()
		m.vp.setLines(m.renderedLines)
		if atBottom {
			m.vp.gotoBottom()
			m.newOutput = false
		} else {
			m.newOutput = true
		}
	}
	return nil
}

func (m *Model) clampCursorToViewport() {
	if len(m.rawLines) == 0 || m.vp.height <= 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	top := m.vp.yOffset
	bottom := min(len(m.rawLines)-1, m.vp.yOffset+m.vp.height-1)
	if m.cursorLine < top {
		m.cursorLine = top
	}
	if m.cursorLine > bottom {
		m.cursorLine = bottom
	}
	m.clampCursorCol()
}

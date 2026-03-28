package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// isEnterKey は Enter キーかどうかを判定する。
// WSL2/Windows Terminal 環境で tea.KeyEnter が正しく認識されない場合の回避策を含む。
func isEnterKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyEnter {
		return true
	}
	// フォールバック: 文字列比較
	s := msg.String()
	return s == "enter" || s == "\r" || s == "\n"
}

// handleKeyMsg はキー入力を処理する。
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tuiDebugf("KeyMsg: Type=%d(%s) Runes=%v String=%q", msg.Type, msg.Type, msg.Runes, msg.String())

	// Ctrl+C は常に最優先
	if msg.Type == tea.KeyCtrlC {
		if m.agent.IsProcessing() {
			m.agent.Cancel()
			m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again to exit.")
			return m, nil
		}
		now := time.Now()
		if !m.lastInterrupt.IsZero() && now.Sub(m.lastInterrupt) < 3*time.Second {
			m.quitting = true
			m.agent.Cleanup()
			return m, tea.Quit
		}
		m.lastInterrupt = now
		m.appendSystemInfo("⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
		return m, nil
	}

	// ナビゲーションモード
	if m.navigationMode {
		return m.handleNavigationKey(msg)
	}

	// 入力モード
	switch {
	case msg.Type == tea.KeyEsc:
		// 入力欄が空の場合のみ NAV モードに入る
		if strings.TrimSpace(m.textInput.Value()) == "" {
			m.navigationMode = true
			m.syncCursorToViewportTop()
			m.textInput.Blur()
			m.chromeDirty = true
			return m, nil
		}

	case isEnterKey(msg):
		tuiDebugf("Enter detected, textInput value=%q", m.textInput.Value())
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}
		m.textInput.Reset()

		m.appendMessage(ChatMessage{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now(),
		})

		if strings.HasPrefix(input, "/") {
			if input == "/exit" || input == "/quit" {
				m.quitting = true
				m.agent.Cleanup()
				return m, tea.Quit
			}
			if m.agent.HandleCommand(input) {
				m.statusLine = m.agent.GetStatusLine()
				return m, nil
			}
		}

		return m, m.sendChat(input)

	case msg.Type == tea.KeyUp:
		m.vp.scrollUp(1)
		m.afterViewportScroll()
		return m, nil
	case msg.Type == tea.KeyDown:
		m.vp.scrollDown(1)
		m.afterViewportScroll()
		return m, nil
	case msg.Type == tea.KeyPgUp:
		m.vp.scrollUp(m.vp.height)
		m.afterViewportScroll()
		return m, nil
	case msg.Type == tea.KeyPgDown:
		m.vp.scrollDown(m.vp.height)
		m.afterViewportScroll()
		return m, nil
	}

	// その他のキーは textInput に渡す
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.chromeDirty = true
	return m, cmd
}

// handleNavigationKey はナビゲーションモードのキー処理。
func (m Model) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.visualMode != visualModeOff {
			m.clearVisualSelection()
			m.chromeDirty = true
			return m, nil
		}
		// ブロックフォーカス中 → フォーカス解除
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
			m.chromeDirty = true
			return m, nil
		}
		// NAVモード終了
		m.clearVisualSelection()
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case tea.KeyEnter:
		// ブロックフォーカス中 → 折りたたみトグル
		if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			m.toggleToolBlock(m.focusedBlock)
			return m, nil
		}
		// NAVモード終了
		m.clearVisualSelection()
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
		}
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case tea.KeyTab:
		if m.visualMode != visualModeOff {
			return m, nil
		}
		// Tab: フォーカス中 → トグル、未フォーカス → 最後のブロックにフォーカス
		if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			m.toggleToolBlock(m.focusedBlock)
		} else if len(m.toolBlocks) > 0 {
			m.setBlockFocus(len(m.toolBlocks) - 1)
			m.chromeDirty = true
		}
		return m, nil
	}

	s := msg.String()

	if m.yPressed {
		m.yPressed = false
		if s == "y" && m.focusedBlock < 0 && m.visualMode == visualModeOff {
			m.copyCursorLine()
			return m, nil
		}
		if m.focusedBlock < 0 && m.visualMode == visualModeOff {
			m.copyDefaultSelectionTarget()
		}
	}

	// g の2回押し判定
	if m.gPressed {
		m.gPressed = false
		if s == "g" && m.focusedBlock < 0 {
			targetLine := 0
			if m.pendingCount > 0 {
				targetLine = min(m.pendingCount-1, max(0, len(m.rawLines)-1))
				m.pendingCount = 0
			}
			m.moveCursorTo(targetLine)
			return m, nil
		}
		// g + 別キー → リセットして通常ナビ処理に落とす
	}

	if m.focusedBlock < 0 && m.visualMode == visualModeOff && len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		m.pendingCount = m.pendingCount*10 + int(s[0]-'0')
		m.chromeDirty = true
		return m, nil
	}
	if m.focusedBlock < 0 && m.visualMode == visualModeOff && len(s) == 1 && s[0] == '0' && m.pendingCount > 0 {
		m.pendingCount *= 10
		m.chromeDirty = true
		return m, nil
	}

	switch s {
	case "q", "i":
		m.clearVisualSelection()
		if m.focusedBlock >= 0 {
			m.clearBlockFocus()
		}
		m.navigationMode = false
		m.resetNavPending()
		m.textInput.Focus()
		m.chromeDirty = true
		return m, nil
	case "j":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock + 1)
		} else {
			m.moveCursorTo(m.cursorLine + m.consumePendingCountOr(1))
		}
	case "k":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock - 1)
		} else {
			m.moveCursorTo(m.cursorLine - m.consumePendingCountOr(1))
		}
	case "h":
		if m.focusedBlock < 0 {
			m.moveCursorCol(-m.consumePendingCountOr(1))
		}
	case "l":
		if m.focusedBlock < 0 {
			m.moveCursorCol(m.consumePendingCountOr(1))
		}
	case "d":
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine + max(1, m.vp.height/2)*m.consumePendingCountOr(1))
		}
	case "u":
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine - max(1, m.vp.height/2)*m.consumePendingCountOr(1))
		}
	case "G":
		if m.focusedBlock < 0 {
			if m.pendingCount > 0 {
				m.moveCursorTo(m.pendingCount - 1)
				m.pendingCount = 0
			} else {
				m.moveCursorTo(len(m.rawLines) - 1)
			}
		}
	case "g":
		if m.focusedBlock < 0 {
			m.gPressed = true
		}
	case "w":
		if m.focusedBlock < 0 {
			m.moveCursorToNextWordStart(m.consumePendingCountOr(1))
		}
	case "b":
		if m.focusedBlock < 0 {
			m.moveCursorToPrevWordStart(m.consumePendingCountOr(1))
		}
	case "e":
		if m.focusedBlock < 0 {
			m.moveCursorToWordEnd(m.consumePendingCountOr(1))
		}
	case "0":
		if m.focusedBlock < 0 {
			m.moveCursorToLineStart(false, 1)
		}
	case "^":
		if m.focusedBlock < 0 {
			m.moveCursorToLineStart(true, m.consumePendingCountOr(1))
		}
	case "$":
		if m.focusedBlock < 0 {
			m.moveCursorToLineEnd(m.consumePendingCountOr(1))
		}
	case "v":
		if m.focusedBlock < 0 {
			m.visualMode = visualModeChar
			m.visualStart = visualPosition{line: m.cursorLine, col: m.cursorCol}
			m.yPressed = false
			m.chromeDirty = true
		}
	case "V":
		if m.focusedBlock < 0 {
			m.visualMode = visualModeLine
			m.visualStart = visualPosition{line: m.cursorLine, col: 0}
			m.yPressed = false
			m.chromeDirty = true
		}
	case "y":
		if m.visualMode != visualModeOff {
			m.copyVisualSelection()
		} else if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
			// フォーカス中のブロック内容をコピー
			content := m.toolBlocks[m.focusedBlock].tool.Detail
			if err := m.agent.CopyText(content); err == nil {
				m.setTransientStatus("✅ Copied block to clipboard")
			} else {
				m.setTransientStatus("Copy failed: " + err.Error())
			}
		} else {
			m.yPressed = true
		}
	default:
		// スクロールキーもサポート
		switch msg.Type {
		case tea.KeyUp:
			if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
				m.moveBlockFocus(m.focusedBlock - 1)
			} else {
				m.moveCursorTo(m.cursorLine - 1)
			}
		case tea.KeyDown:
			if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
				m.moveBlockFocus(m.focusedBlock + 1)
			} else {
				m.moveCursorTo(m.cursorLine + 1)
			}
		case tea.KeyPgUp:
			if m.focusedBlock < 0 {
				m.moveCursorTo(m.cursorLine - m.vp.height)
			}
		case tea.KeyPgDown:
			if m.focusedBlock < 0 {
				m.moveCursorTo(m.cursorLine + m.vp.height)
			}
		}
	}
	return m, nil
}

// sendChat は goroutine で agent.Chat を呼び出す tea.Cmd を返す。
func (m Model) sendChat(input string) tea.Cmd {
	return func() tea.Msg {
		m.agent.Chat(input)
		return AgentDoneMsg{}
	}
}

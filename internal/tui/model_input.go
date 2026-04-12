package tui

import (
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

var readClipboardText = clipboard.ReadAll

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

func isBackspaceKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH {
		return true
	}
	s := msg.String()
	return s == "backspace" || s == "ctrl+h"
}

// handleKeyMsg はキー入力を処理する。
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tuiDebugf("KeyMsg: Type=%d(%s) Runes=%v String=%q", msg.Type, msg.Type, msg.Runes, msg.String())

	// Ctrl+C は常に最優先
	if msg.Type == tea.KeyCtrlC {
		return m.handleCtrlC()
	}

	if msg.Paste {
		m.switchToComposerInput()
		m.handleComposerPaste(string(msg.Runes))
		return m, nil
	}

	if key.Matches(msg, m.textInput.KeyMap.Paste) {
		return m.handleClipboardPaste()
	}

	if m.navigationMode {
		return m.handleNavigationKey(msg)
	}

	return m.handleComposerInputKey(msg)
}

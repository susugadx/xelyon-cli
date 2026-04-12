package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderInputDock は入力欄部分（パディング行+入力行+パディング行）を構築する。
func (m *Model) renderInputDock() string {
	const inputBg = "\033[48;5;236m"
	const textFg = "\033[38;5;252m"
	const pasteFg = "\033[38;5;244m"
	const pasteID = "\033[38;5;81m"
	tiView := strings.ReplaceAll(m.textInput.View(), "\033[0m", "\033[0m"+inputBg)
	inputLine := fillANSITextWidth(inputBg+" \033[38;5;46m"+inputPrompt+"\033[38;5;252m"+tiView+"\033[0m", m.width, inputBg)
	rows := m.visibleComposerRows()
	lines := make([]string, 0, len(rows)+inputHeight)
	for _, row := range rows {
		switch row.kind {
		case composerPartText:
			lines = append(lines, fillANSITextWidth(inputBg+" "+textFg+m.formatComposerTextRow(row.text)+"\033[0m", m.width, inputBg))
		case composerPartPaste:
			summary := strings.Replace(
				m.formatPasteBlockSummary(row.pasteBlock),
				"#",
				pasteID+"#",
				1,
			)
			lines = append(lines, fillANSITextWidth(inputBg+" "+pasteFg+summary+"\033[0m", m.width, inputBg))
		}
	}
	lines = append(lines, m.padLineCache, inputLine, m.padLineCache)
	return strings.Join(lines, "\n")
}

// renderStatusBar はステータスバー行を構築する。
func (m *Model) renderStatusBar() string {
	const hintColor = "\033[38;5;244m"
	statusLine := sanitizeSingleLineANSI(m.statusLine)

	var statusText string
	if m.navigationMode {
		statusText = " \033[48;5;33;38;5;255m NAV \033[0m " + statusLine
	} else if m.agent.IsProcessing() {
		statusText = " " + m.spinner.View() + " " + statusLine
	} else {
		statusText = " " + statusLine
	}

	if m.newOutput && !m.vp.atBottom() {
		statusText += "  \033[48;5;63;38;5;230m ↓ New output \033[0m"
	}

	if m.transientStatus != "" && time.Now().Before(m.transientStatusUntil) {
		statusText += "  \033[38;5;82m" + sanitizeSingleLineANSI(m.transientStatus) + "\033[0m"
	}

	hints := statusHintsNormal
	if m.hasActiveMouseSelection() || m.mouseDragging {
		hints = statusHintsMouseSel
	} else if m.navigationMode {
		if m.visualMode == visualModeChar {
			hints = statusHintsVisual
		} else if m.visualMode == visualModeLine {
			hints = statusHintsVisualLine
		} else if m.focusedBlock >= 0 {
			hints = statusHintsBlockFocus
		} else {
			hints = statusHintsNav
		}
	}
	statusBar := statusText
	for _, hint := range hints {
		padding := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint)
		if padding >= 2 {
			statusBar = statusText + strings.Repeat(" ", padding) + hintColor + hint + "\033[0m"
			break
		}
	}
	return fitANSITextWidth(statusBar, m.width)
}

// rebuildChrome は入力欄+ステータスバーを再構築する。
// Update() 内で chromeDirty 時のみ呼ばれる（View() は値レシーバーなので書き込み不可）。
func (m *Model) rebuildChrome() {
	m.chromeCache = m.renderInputDock() + "\n" + m.renderStatusBar()
}

// View は bubbletea の View を実装する。
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}
	if m.screen == screenConfig {
		return m.configView()
	}
	return m.viewportView() + "\n" + m.chromeCache
}

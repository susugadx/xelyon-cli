package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tuicomposer "github.com/susugadx/xelyon-cli/internal/tui/composer"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderInputDock は入力欄部分（パディング行+入力行+パディング行）を構築する。
func (m *Model) renderInputDock() string {
	chrome := theme.Chrome
	tiView := strings.ReplaceAll(m.textInput.View(), chrome.Reset, chrome.Reset+chrome.InputBg)
	inputLine := termtext.FillANSITextWidth(chrome.InputBg+" "+chrome.InputPrompt+inputPrompt+chrome.InputTextFg+tiView+chrome.Reset, m.width, chrome.InputBg)
	rows := m.visibleComposerRows()
	attStart, attEnd := m.visibleAttachmentRange()
	attachmentCount := attEnd - attStart
	suggestionRows := m.renderSlashSuggestionRows()
	lines := make([]string, 0, len(suggestionRows)+attachmentCount+len(rows)+inputHeight)
	lines = append(lines, suggestionRows...)
	for i := 0; i < attachmentCount; i++ {
		att := m.attachments[attStart+i]
		summary := strings.Replace(
			m.formatAttachmentSummary(att, attStart+i+1),
			"#",
			chrome.InputPasteID+"#",
			1,
		)
		lines = append(lines, termtext.FillANSITextWidth(chrome.InputBg+" "+chrome.InputPasteFg+summary+chrome.Reset, m.width, chrome.InputBg))
	}
	for _, row := range rows {
		switch row.Kind {
		case tuicomposer.PartText:
			lines = append(lines, termtext.FillANSITextWidth(chrome.InputBg+" "+chrome.InputTextFg+m.formatComposerTextRow(row.Text)+chrome.Reset, m.width, chrome.InputBg))
		case tuicomposer.PartPaste:
			summary := strings.Replace(
				m.formatPasteBlockSummary(row.PasteBlock),
				"#",
				chrome.InputPasteID+"#",
				1,
			)
			lines = append(lines, termtext.FillANSITextWidth(chrome.InputBg+" "+chrome.InputPasteFg+summary+chrome.Reset, m.width, chrome.InputBg))
		}
	}
	lines = append(lines, m.padLineCache, inputLine, m.padLineCache)
	return strings.Join(lines, "\n")
}

// renderStatusBar はステータスバー行を構築する。
func (m *Model) renderStatusBar() string {
	chrome := theme.Chrome
	statusLine := termtext.SanitizeSingleLineANSI(m.statusLine)

	var statusText string
	if m.navigationMode {
		statusText = " " + chrome.NavBadge + " NAV " + chrome.Reset + " " + statusLine
	} else if m.conversation.IsProcessing() {
		statusText = " " + m.spinner.View() + " " + statusLine
	} else {
		statusText = " " + statusLine
	}

	if m.newOutput && !m.vp.atBottom() {
		statusText += "  " + chrome.NewOutput + " ↓ New output " + chrome.Reset
	}

	if m.transientStatus != "" && time.Now().Before(m.transientStatusUntil) {
		statusText += "  " + chrome.SuccessFg + termtext.SanitizeSingleLineANSI(m.transientStatus) + chrome.Reset
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
			statusBar = statusText + strings.Repeat(" ", padding) + chrome.HintFg + hint + chrome.Reset
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
	base := m.baseView()
	if m.prompt != nil {
		return m.renderPromptOverlay(base)
	}
	return base
}

func (m Model) baseView() string {
	if m.screen == screenConfig {
		return m.configView()
	}
	if m.screen == screenReview {
		return m.reviewView()
	}
	if m.screen == screenProject {
		return m.projectView()
	}
	return m.viewportView() + "\n" + m.chromeCache
}

package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m *Model) buildStatusBarLine() string {
	statusText := m.buildStatusText(time.Now())
	hints := m.activeStatusHints()
	return m.composeStatusBarLine(statusText, hints)
}

func (m *Model) buildStatusText(now time.Time) string {
	chrome := theme.Chrome
	statusLine := termtext.SanitizeSingleLineANSI(m.statusLine)

	statusText := " " + chrome.StatusFg + statusLine + chrome.Reset
	if m.navigationMode {
		statusText = " " + chrome.NavBadge + " NAV " + chrome.Reset + chrome.StatusSepFg + " " + chrome.Reset + chrome.StatusFg + statusLine + chrome.Reset
	} else if m.conversation.IsProcessing() {
		statusText = " " + m.spinner.View() + chrome.StatusSepFg + " " + chrome.Reset + chrome.StatusFg + statusLine + chrome.Reset
	}
	if m.newOutput && !m.vp.atBottom() {
		statusText += chrome.StatusSepFg + "  " + chrome.Reset + chrome.NewOutput + " ↓ New output " + chrome.Reset
	}
	if m.transientStatus != "" && now.Before(m.transientStatusUntil) {
		statusText += chrome.StatusSepFg + "  " + chrome.Reset + chrome.SuccessFg + termtext.SanitizeSingleLineANSI(m.transientStatus) + chrome.Reset
	}
	return statusText
}

func (m Model) activeStatusHints() []string {
	if m.hasActiveMouseSelection() || m.mouseDragging {
		return statusHintsMouseSel
	}
	if !m.navigationMode {
		return statusHintsNormal
	}
	if m.visualMode == visualModeChar {
		return statusHintsVisual
	}
	if m.visualMode == visualModeLine {
		return statusHintsVisualLine
	}
	if m.focusedBlock >= 0 {
		return statusHintsBlockFocus
	}
	return statusHintsNav
}

func (m Model) composeStatusBarLine(statusText string, hints []string) string {
	statusBar := statusText
	for _, hint := range hints {
		if line, ok := m.composeStatusBarWithWorkingDir(statusText, hint); ok {
			return line
		}
	}
	for _, hint := range hints {
		if line, ok := m.composeStatusBarWithoutWorkingDir(statusText, hint); ok {
			statusBar = line
			break
		}
	}
	return fitANSITextWidth(statusBar, m.width)
}

func (m Model) composeStatusBarWithWorkingDir(statusText, hint string) (string, bool) {
	pathMaxWidth := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint) - 4
	workingDir := m.renderWorkingDirStatusSegment(pathMaxWidth)
	if workingDir == "" {
		return "", false
	}
	chrome := theme.Chrome
	statusWithPath := statusText + chrome.StatusSepFg + "  " + chrome.Reset + workingDir
	padding := m.width - lipgloss.Width(statusWithPath) - lipgloss.Width(hint)
	if padding < 2 {
		return "", false
	}
	line := statusWithPath + strings.Repeat(" ", padding) + chrome.HintFg + hint + chrome.Reset
	return fitANSITextWidth(line, m.width), true
}

func (m Model) composeStatusBarWithoutWorkingDir(statusText, hint string) (string, bool) {
	padding := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint)
	if padding < 2 {
		return "", false
	}
	chrome := theme.Chrome
	return statusText + strings.Repeat(" ", padding) + chrome.HintFg + hint + chrome.Reset, true
}

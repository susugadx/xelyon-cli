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

type statusTextSegment struct {
	text string
}

type statusBarLayoutRequest struct {
	width      int
	statusText string
	hints      []string
}

type statusBarLineParts struct {
	statusText string
	workingDir string
	hint       string
}

func (m *Model) buildStatusText(now time.Time) string {
	return renderStatusTextSegments(m.statusTextSegments(now))
}

func (m *Model) statusTextSegments(now time.Time) []statusTextSegment {
	chrome := theme.Chrome
	statusLine := termtext.SanitizeSingleLineANSI(m.statusLine)

	segments := []statusTextSegment{{text: m.primaryStatusTextSegment(statusLine)}}
	if m.newOutput && !m.vp.atBottom() {
		segments = append(segments, statusTextSegment{
			text: chrome.StatusSepFg + "  " + chrome.Reset + chrome.NewOutput + " ↓ New output " + chrome.Reset,
		})
	}
	if m.transientStatus != "" && now.Before(m.transientStatusUntil) {
		segments = append(segments, statusTextSegment{
			text: chrome.StatusSepFg + "  " + chrome.Reset + chrome.SuccessFg + termtext.SanitizeSingleLineANSI(m.transientStatus) + chrome.Reset,
		})
	}
	return segments
}

func (m *Model) primaryStatusTextSegment(statusLine string) string {
	chrome := theme.Chrome
	if m.navigationMode {
		return " " + chrome.NavBadge + " NAV " + chrome.Reset + chrome.StatusSepFg + " " + chrome.Reset + chrome.StatusFg + statusLine + chrome.Reset
	}
	if m.conversation.IsProcessing() {
		return " " + m.spinner.View() + chrome.StatusSepFg + " " + chrome.Reset + chrome.StatusFg + statusLine + chrome.Reset
	}
	return " " + chrome.StatusFg + statusLine + chrome.Reset
}

func renderStatusTextSegments(segments []statusTextSegment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.text)
	}
	return b.String()
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
	return m.fitStatusBarLine(statusBarLayoutRequest{
		width:      m.width,
		statusText: statusText,
		hints:      hints,
	})
}

func (m Model) fitStatusBarLine(req statusBarLayoutRequest) string {
	statusBar := req.statusText
	for _, hint := range req.hints {
		if line, ok := m.composeStatusBarWithWorkingDir(req.statusText, hint); ok {
			return line
		}
	}
	for _, hint := range req.hints {
		if line, ok := m.composeStatusBarWithoutWorkingDir(req.statusText, hint); ok {
			statusBar = line
			break
		}
	}
	return fitANSITextWidth(statusBar, req.width)
}

func (m Model) composeStatusBarWithWorkingDir(statusText, hint string) (string, bool) {
	pathMaxWidth := m.width - lipgloss.Width(statusText) - lipgloss.Width(hint) - 4
	workingDir := m.renderWorkingDirStatusSegment(pathMaxWidth)
	if workingDir == "" {
		return "", false
	}
	return renderStatusBarLineParts(statusBarLineParts{
		statusText: statusText,
		workingDir: workingDir,
		hint:       hint,
	}, m.width)
}

func (m Model) composeStatusBarWithoutWorkingDir(statusText, hint string) (string, bool) {
	return renderStatusBarLineParts(statusBarLineParts{
		statusText: statusText,
		hint:       hint,
	}, m.width)
}

func renderStatusBarLineParts(parts statusBarLineParts, width int) (string, bool) {
	chrome := theme.Chrome
	statusWithPath := parts.statusText
	if parts.workingDir != "" {
		statusWithPath += chrome.StatusSepFg + "  " + chrome.Reset + parts.workingDir
	}
	padding := width - lipgloss.Width(statusWithPath) - lipgloss.Width(parts.hint)
	if padding < 2 {
		return "", false
	}
	line := statusWithPath + strings.Repeat(" ", padding) + chrome.HintFg + parts.hint + chrome.Reset
	if parts.workingDir != "" {
		line = fitANSITextWidth(line, width)
	}
	return line, true
}

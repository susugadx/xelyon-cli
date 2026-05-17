package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m *Model) buildStatusBarLine() string {
	now := time.Now()
	segments := m.statusTextSegments(now)
	hints := m.activeStatusHints()
	return m.composeStatusBarLine(segments, hints)
}

type statusTextSegment struct {
	text string
	key  statusTextSegmentKey
}

type statusTextSegmentKey string

const (
	statusSegmentPrimary   statusTextSegmentKey = "primary"
	statusSegmentProvider  statusTextSegmentKey = "provider"
	statusSegmentMode      statusTextSegmentKey = "mode"
	statusSegmentTokens    statusTextSegmentKey = "tokens"
	statusSegmentCost      statusTextSegmentKey = "cost"
	statusSegmentNewOutput statusTextSegmentKey = "new_output"
	statusSegmentTransient statusTextSegmentKey = "transient"
)

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
	snapshot := sanitizeStatusSnapshot(m.statusSnapshot)
	if statusLine != "" && statusLine != snapshot.LegacyLine {
		snapshot.Mode = statusLine
		snapshot.LegacyLine = statusLine
	}

	var segments []statusTextSegment
	if m.conversation.IsProcessing() {
		segments = append(segments, statusTextSegment{text: m.spinner.View(), key: statusSegmentPrimary})
		if providerModel := providerModelStatusText(snapshot, statusLine); providerModel != "" {
			segments = append(segments, statusTextSegment{text: chrome.StatusFg + providerModel + chrome.Reset, key: statusSegmentProvider})
		}
		if mode := processingModeStatusText(snapshot); mode != "" {
			segments = append(segments, statusTextSegment{text: chrome.StatusFg + mode + chrome.Reset, key: statusSegmentMode})
		}
		if snapshot.Tokens != "" {
			segments = append(segments, statusTextSegment{text: chrome.StatusFg + snapshot.Tokens + " tok" + chrome.Reset, key: statusSegmentTokens})
		}
		if snapshot.Cost != "" {
			segments = append(segments, statusTextSegment{text: chrome.StatusFg + snapshot.Cost + chrome.Reset, key: statusSegmentCost})
		}
		return segments
	}

	segments = append(segments, statusTextSegment{text: m.primaryStatusTextSegment(snapshot, statusLine), key: statusSegmentPrimary})
	if m.newOutput && !m.vp.atBottom() {
		segments = append(segments, statusTextSegment{
			text: chrome.NewOutput + "↓ New output" + chrome.Reset,
			key:  statusSegmentNewOutput,
		})
	}
	if m.transientStatus != "" && now.Before(m.transientStatusUntil) {
		segments = append(segments, statusTextSegment{
			text: chrome.SuccessFg + termtext.SanitizeSingleLineANSI(m.transientStatus) + chrome.Reset,
			key:  statusSegmentTransient,
		})
	}
	return segments
}

func (m *Model) primaryStatusTextSegment(snapshot StatusSnapshot, statusLine string) string {
	chrome := theme.Chrome
	modeText := idlePrimaryStatusText(snapshot, statusLine)
	if m.navigationMode {
		return chrome.NavBadge + " NAV " + chrome.Reset + chrome.StatusFg + modeText + chrome.Reset
	}
	return chrome.StatusFg + modeText + chrome.Reset
}

func idlePrimaryStatusText(snapshot StatusSnapshot, statusLine string) string {
	if snapshot.LegacyLine != "" && snapshot.LegacyLine != snapshot.Mode {
		return snapshot.LegacyLine
	}
	if statusLine != "" && statusLine != snapshot.Mode {
		return statusLine
	}
	if snapshot.Mode != "" {
		return snapshot.Mode
	}
	return statusLine
}

func renderStatusTextSegments(segments []statusTextSegment) string {
	var b strings.Builder
	chrome := theme.Chrome
	for i, segment := range segments {
		if i == 0 {
			b.WriteString(" ")
		} else {
			b.WriteString(chrome.StatusSepFg)
			b.WriteString(" | ")
			b.WriteString(chrome.Reset)
		}
		b.WriteString(segment.text)
	}
	return b.String()
}

func sanitizeStatusSnapshot(snapshot StatusSnapshot) StatusSnapshot {
	snapshot.Provider = termtext.SanitizeSingleLineANSI(snapshot.Provider)
	snapshot.Model = termtext.SanitizeSingleLineANSI(snapshot.Model)
	snapshot.Mode = termtext.SanitizeSingleLineANSI(snapshot.Mode)
	snapshot.Tokens = termtext.SanitizeSingleLineANSI(snapshot.Tokens)
	snapshot.Cost = termtext.SanitizeSingleLineANSI(snapshot.Cost)
	snapshot.LegacyLine = termtext.SanitizeSingleLineANSI(snapshot.LegacyLine)
	return snapshot
}

func providerModelStatusText(snapshot StatusSnapshot, fallback string) string {
	switch {
	case snapshot.Provider != "" && snapshot.Model != "":
		return snapshot.Provider + "/" + snapshot.Model
	case snapshot.Model != "":
		return snapshot.Model
	case snapshot.Provider != "":
		return snapshot.Provider
	default:
		return fallback
	}
}

func processingModeStatusText(snapshot StatusSnapshot) string {
	mode := strings.TrimSpace(snapshot.Mode)
	if mode == "" || mode == strings.TrimSpace(snapshot.LegacyLine) {
		return ""
	}
	return mode
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

func (m Model) composeStatusBarLine(segments []statusTextSegment, hints []string) string {
	return m.fitStatusBarLine(statusBarLayoutRequest{
		width:      m.width,
		statusText: renderStatusTextSegments(segments),
		hints:      hints,
	}, segments)
}

func (m Model) fitStatusBarLine(req statusBarLayoutRequest, segments ...[]statusTextSegment) string {
	if len(segments) > 0 {
		if line := m.fitStatusBarSegments(req, segments[0]); line != "" {
			return line
		}
	}
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

func (m Model) fitStatusBarSegments(req statusBarLayoutRequest, segments []statusTextSegment) string {
	dropOrder := []statusTextSegmentKey{
		statusSegmentCost,
		statusSegmentTokens,
		statusSegmentNewOutput,
		statusSegmentTransient,
		statusSegmentMode,
	}
	for dropCount := 0; dropCount <= len(dropOrder)+1; dropCount++ {
		dropped := make(map[statusTextSegmentKey]struct{}, dropCount)
		for i := 0; i < dropCount && i < len(dropOrder); i++ {
			dropped[dropOrder[i]] = struct{}{}
		}
		dropCWD := dropCount > len(dropOrder)
		statusText := renderStatusTextSegments(filterStatusSegments(segments, dropped))
		for _, hint := range req.hints {
			if !dropCWD {
				if line, ok := m.composeStatusBarWithWorkingDir(statusText, hint); ok {
					return line
				}
				continue
			}
			if line, ok := m.composeStatusBarWithoutWorkingDir(statusText, hint); ok {
				return line
			}
		}
	}
	return ""
}

func filterStatusSegments(segments []statusTextSegment, dropped map[statusTextSegmentKey]struct{}) []statusTextSegment {
	if len(dropped) == 0 {
		return segments
	}
	out := make([]statusTextSegment, 0, len(segments))
	for _, segment := range segments {
		if _, ok := dropped[segment.key]; ok {
			continue
		}
		out = append(out, segment)
	}
	return out
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

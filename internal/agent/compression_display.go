package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
	displayui "github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	compressionDisplayToolName = displayui.ToolNameCompress

	compressionDisplayModeHistory    = displayui.ToolCompressionModeHistory
	compressionDisplayModeCompactAPI = displayui.ToolCompressionModeCompactAPI

	compressionDisplayReasonManual     = displayui.ToolCompressionReasonManual
	compressionDisplayReasonAuto       = displayui.ToolCompressionReasonAuto
	compressionDisplayReasonTokenLimit = displayui.ToolCompressionReasonTokenLimit
)

type compressionDisplayOperation struct {
	id           string
	startedAt    time.Time
	mode         string
	reason       string
	keepRecent   int
	beforeTokens int
	active       bool
}

func (a *Agent) usesTUICompressionDisplay() bool {
	return a != nil && a.tuiToolResultCh != nil && !a.tuiToolResultClosed.Load()
}

func (a *Agent) shouldPrintCompressionOutput() bool {
	return !a.usesTUICompressionDisplay()
}

func (a *Agent) beginCompressionDisplay(mode, reason string, keepRecent, beforeTokens int) compressionDisplayOperation {
	startedAt := time.Now()
	op := compressionDisplayOperation{
		id:           fmt.Sprintf("compress-%d", startedAt.UnixNano()),
		startedAt:    startedAt,
		mode:         normalizeCompressionDisplayMode(mode),
		reason:       normalizeCompressionDisplayReason(reason),
		keepRecent:   keepRecent,
		beforeTokens: beforeTokens,
		active:       a.usesTUICompressionDisplay(),
	}
	if !op.active {
		return op
	}
	a.emitTUIToolInfo(tools.ToolResultInfo{
		ToolName:  compressionDisplayToolName,
		Args:      op.args(0, ""),
		Result:    op.detail(0, ""),
		ID:        op.id,
		Status:    tools.ToolStatusRunning,
		StartedAt: op.startedAt,
	})
	return op
}

func (a *Agent) updateCompressionDisplay(op compressionDisplayOperation, mode, result string) compressionDisplayOperation {
	if !op.active {
		return op
	}
	op.mode = normalizeCompressionDisplayMode(mode)
	a.emitTUIToolInfo(tools.ToolResultInfo{
		ToolName:  compressionDisplayToolName,
		Args:      op.args(0, ""),
		Result:    strings.TrimSpace(result),
		ID:        op.id,
		Status:    tools.ToolStatusRunning,
		StartedAt: op.startedAt,
	})
	return op
}

func (a *Agent) finishCompressionDisplay(op compressionDisplayOperation, afterTokens int, err error) {
	if !op.active {
		return
	}
	status := tools.ToolStatusOK
	result := op.detail(afterTokens, "")
	if err != nil {
		status = tools.ToolStatusError
		result = err.Error()
	}
	a.emitTUIToolInfo(tools.ToolResultInfo{
		ToolName:  compressionDisplayToolName,
		Args:      op.args(afterTokens, ""),
		Result:    result,
		Error:     err != nil,
		ID:        op.id,
		Status:    status,
		StartedAt: op.startedAt,
		Duration:  time.Since(op.startedAt),
	})
}

func (a *Agent) finishCompressionDisplaySkipped(op compressionDisplayOperation, reason string) {
	if !op.active {
		return
	}
	a.emitTUIToolInfo(tools.ToolResultInfo{
		ToolName:  compressionDisplayToolName,
		Args:      op.args(0, reason),
		Result:    op.detail(0, reason),
		ID:        op.id,
		Status:    tools.ToolStatusOK,
		StartedAt: op.startedAt,
		Duration:  time.Since(op.startedAt),
	})
}

func (op compressionDisplayOperation) args(afterTokens int, outcome string) map[string]string {
	args := map[string]string{
		displayui.ToolArgCompressionMode:   op.mode,
		displayui.ToolArgCompressionReason: op.reason,
	}
	if op.keepRecent > 0 {
		args[displayui.ToolArgCompressionKeepRecent] = fmt.Sprintf("%d", op.keepRecent)
	}
	if op.beforeTokens > 0 {
		args[displayui.ToolArgCompressionBeforeTokens] = formatNumber(op.beforeTokens)
	}
	if afterTokens > 0 {
		args[displayui.ToolArgCompressionAfterTokens] = formatNumber(afterTokens)
	}
	if strings.TrimSpace(outcome) != "" {
		args[displayui.ToolArgCompressionOutcome] = strings.TrimSpace(outcome)
	}
	return args
}

func (op compressionDisplayOperation) detail(afterTokens int, outcome string) string {
	lines := []string{
		"Context compression",
		"Mode: " + op.mode,
		"Reason: " + op.reason,
	}
	if op.beforeTokens > 0 {
		lines = append(lines, "Before tokens: "+formatNumber(op.beforeTokens))
	}
	if afterTokens > 0 {
		lines = append(lines, "After tokens: "+formatNumber(afterTokens))
	}
	if op.keepRecent > 0 {
		lines = append(lines, fmt.Sprintf("Kept recent messages: %d", op.keepRecent))
	}
	if strings.TrimSpace(outcome) != "" {
		lines = append(lines, "Outcome: "+strings.TrimSpace(outcome))
	}
	return strings.Join(lines, "\n")
}

func normalizeCompressionDisplayMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case compressionDisplayModeCompactAPI:
		return compressionDisplayModeCompactAPI
	default:
		return compressionDisplayModeHistory
	}
}

func normalizeCompressionDisplayReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case compressionDisplayReasonAuto:
		return compressionDisplayReasonAuto
	case compressionDisplayReasonTokenLimit:
		return compressionDisplayReasonTokenLimit
	default:
		return compressionDisplayReasonManual
	}
}

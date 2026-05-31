package tuiagent

import (
	"context"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

func (a *TUIAdapter) reviewProgressSink(ctx context.Context) review.ReviewProgressSink {
	if a == nil || (a.sendReviewProgress == nil && a.sendToolResult == nil) {
		return nil
	}
	runID, hasRunID := tui.ReviewRunIDFromContext(ctx)
	sendReviewProgress := a.sendReviewProgress
	sendToolResult := a.sendToolResult
	return func(event review.ReviewProgressEvent) {
		tool, ok := reviewProgressToolResult(event)
		if !ok {
			return
		}
		if hasRunID {
			if sendReviewProgress != nil {
				sendReviewProgress(tui.ReviewProgressMsg{RunID: runID, Tool: tool})
			}
			return
		}
		if sendToolResult == nil {
			return
		}
		sendToolResult(tui.AppendToolResultMsg{Tool: tool})
	}
}

func reviewProgressToolResult(event review.ReviewProgressEvent) (tui.ToolResult, bool) {
	if strings.TrimSpace(event.ID) == "" && event.Phase == "" {
		return tui.ToolResult{}, false
	}

	name := strings.TrimSpace(event.Label)
	if name == "" {
		name = strings.ReplaceAll(string(event.Phase), "_", " ")
	}
	target := reviewProgressToolTarget(event)
	status := reviewProgressToolStatus(event.Status)
	tool := tui.ToolResult{
		Name:             name,
		Summary:          formatTUIToolSummary(status, name, target, event.Duration),
		Detail:           strings.TrimSpace(event.Detail),
		Collapsed:        true,
		Error:            status == tui.ToolStatusError,
		NonBlockingError: event.Phase == review.ReviewProgressPhaseProbe,
		ID:               "review:" + reviewProgressEventID(event),
		Status:           status,
		Target:           target,
		Duration:         event.Duration,
	}
	if status == tui.ToolStatusRunning {
		tool.StartedAt = time.Now()
	}
	return tool, true
}

func reviewProgressEventID(event review.ReviewProgressEvent) string {
	id := strings.TrimSpace(event.ID)
	if id != "" {
		return id
	}
	return string(event.Phase)
}

func reviewProgressToolTarget(event review.ReviewProgressEvent) string {
	detail := strings.TrimSpace(event.Detail)
	if detail == "" {
		return ""
	}
	return "· " + detail
}

func reviewProgressToolStatus(status review.ReviewProgressStatus) tui.ToolStatus {
	switch status {
	case review.ReviewProgressRunning:
		return tui.ToolStatusRunning
	case review.ReviewProgressError:
		return tui.ToolStatusError
	default:
		return tui.ToolStatusOK
	}
}

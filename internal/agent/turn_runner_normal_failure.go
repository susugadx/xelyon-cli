package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type normalModeFailureHandler struct {
	runner           *TurnRunner
	retry            *retryState
	lastFailedResult string
}

func newNormalModeFailureHandler(r *TurnRunner, rs *retryState, lastFailedResult string) *normalModeFailureHandler {
	return &normalModeFailureHandler{
		runner:           r,
		retry:            rs,
		lastFailedResult: lastFailedResult,
	}
}

func (h *normalModeFailureHandler) Handle() error {
	if h == nil || h.retry == nil {
		return nil
	}
	if h.lastFailedResult == "" {
		h.handleRetrySuccess()
		return nil
	}

	level := h.retry.recordFailure(h.lastFailedResult)
	switch level {
	case stalledNone:
		h.handleRetry()
	case stalledSoft:
		h.handleStrategyChangeRetry()
	default:
		h.handleStalledHard()
	}
	return nil
}

func (h *normalModeFailureHandler) handleRetry() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	red.Fprintf(a.output(), "❌ Failed (retry %d)\n", h.retry.count)
	yellow.Fprintf(a.output(), "🔄 Retrying...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildNormalModeRetryMessage(h.retry.count, h.lastFailedResult),
	})
}

func (h *normalModeFailureHandler) handleStrategyChangeRetry() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	yellow.Fprintf(a.output(), "⚠️  Similar failure repeated %d times (retry %d)\n", h.retry.sameCount, h.retry.count)
	yellow.Fprintf(a.output(), "🔄 Retrying with strategy change...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildNormalModeStrategyChangeMessage(h.retry.count, h.retry.sameCount, h.lastFailedResult),
	})
}

func (h *normalModeFailureHandler) handleStalledHard() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	red.Fprintf(a.output(), "❌ Stalled — same error %d times\n", h.retry.sameCount)
	yellow.Fprintln(a.output(), "Could not complete the task automatically. Letting AI respond...")
	h.retry.reset()
}

func (h *normalModeFailureHandler) handleRetrySuccess() {
	a := h.runner.agent
	if h.retry.count <= 0 {
		return
	}
	green.Fprintf(a.output(), "✅ Succeeded (on retry %d)\n", h.retry.count)
	h.retry.reset()
}

func buildNormalModeRetryMessage(attempt int, lastFailedResult string) string {
	return fmt.Sprintf("The previous tool execution FAILED (attempt %d):\n\n%s\n\n%s",
		attempt, lastFailedResult, normalModeRetryInstruction(attempt))
}

func buildNormalModeStrategyChangeMessage(attempt, sameCount int, lastFailedResult string) string {
	return fmt.Sprintf("The previous tool execution FAILED (attempt %d):\n\n%s\n\n"+
		"WARNING: A similar failure has now occurred %d times in a row.\n"+
		"Your previous approach is likely wrong — do not repeat the same fix pattern.\n\n%s",
		attempt, lastFailedResult, sameCount, normalModeRetryInstruction(attempt))
}

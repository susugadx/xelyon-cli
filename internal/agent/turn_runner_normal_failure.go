package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

type normalModeFailureHandler struct {
	runner           *TurnRunner
	retry            *turnsupport.RetryState
	lastFailedResult string
}

func newNormalModeFailureHandler(r *TurnRunner, rs *turnsupport.RetryState, lastFailedResult string) *normalModeFailureHandler {
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

	level := h.retry.RecordFailure(h.lastFailedResult)
	switch level {
	case turnsupport.StalledNone:
		h.handleRetry()
	case turnsupport.StalledSoft:
		h.handleStrategyChangeRetry()
	default:
		h.handleStalledHard()
	}
	return nil
}

func (h *normalModeFailureHandler) handleRetry() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	red.Fprintf(a.output(), "❌ Failed (retry %d)\n", h.retry.Count())
	yellow.Fprintf(a.output(), "🔄 Retrying...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildNormalModeRetryMessage(h.retry.Count(), h.lastFailedResult),
	})
}

func (h *normalModeFailureHandler) handleStrategyChangeRetry() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	yellow.Fprintf(a.output(), "⚠️  Similar failure repeated %d times (retry %d)\n", h.retry.SameCount(), h.retry.Count())
	yellow.Fprintf(a.output(), "🔄 Retrying with strategy change...\n")

	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: buildNormalModeStrategyChangeMessage(h.retry.Count(), h.retry.SameCount(), h.lastFailedResult),
	})
}

func (h *normalModeFailureHandler) handleStalledHard() {
	a := h.runner.agent
	a.ui().ResetTerminalState()
	red.Fprintf(a.output(), "❌ Stalled — same error %d times\n", h.retry.SameCount())
	yellow.Fprintln(a.output(), "Could not complete the task automatically. Letting AI respond...")
	h.retry.Reset()
}

func (h *normalModeFailureHandler) handleRetrySuccess() {
	a := h.runner.agent
	if h.retry.Count() <= 0 {
		return
	}
	green.Fprintf(a.output(), "✅ Succeeded (on retry %d)\n", h.retry.Count())
	h.retry.Reset()
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

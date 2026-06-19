package review

import (
	"strings"
	"time"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

func (r *ReviewRunner) probeResultPromptContextOptions() reviewmodelinput.ProbeResultPromptContextOptions {
	if r == nil {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	mode := normalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == ReviewPromptReductionModeOff {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(mode)
	}
	return reviewmodelinput.ProbeResultPromptContextOptions{
		CommandOutputCompactor: newReviewPromptCommandOutputCompactor(mode, r.promptReductionStats),
	}
}

func (r *ReviewRunner) resetPromptReductionStats() {
	if r == nil {
		return
	}
	r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	r.promptReductionState = &ReviewPromptReductionState{
		Mode:   normalizeReviewPromptReductionMode(r.promptReductionMode),
		Report: r.promptReductionStats.reportValue(),
	}
}

func normalizeReviewRunID(id string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return "review-" + time.Now().UTC().Format("20060102T150405.000000000Z")
}

func (r *ReviewRunner) recordPromptReductionItem(item ReviewPromptReductionItem) {
	if r == nil || normalizeReviewPromptReductionMode(r.promptReductionMode) == ReviewPromptReductionModeOff {
		return
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &ReviewPromptReductionState{Mode: normalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Items = append(r.promptReductionState.Items, item)
	if r.promptReductionStats != nil {
		r.promptReductionStats.recordItem(item)
		r.promptReductionState.Report = r.promptReductionStats.reportValue()
	}
}

// PromptReductionReport は直近 Run の review prompt 削減集計を返す。
func (r *ReviewRunner) PromptReductionReport() ReviewPromptReductionReport {
	if r == nil {
		return ReviewPromptReductionReport{}
	}
	return r.promptReductionStats.reportValue()
}

func (r *ReviewRunner) reviewStateSummaryPrompt(input reviewStateSummaryInput) string {
	if r == nil || normalizeReviewPromptReductionMode(r.promptReductionMode) != ReviewPromptReductionModeApply {
		return ""
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	}
	summary := buildReviewStateSummary(input)
	text := summary.PromptText()
	if strings.TrimSpace(text) == "" {
		return ""
	}
	absorbedCount := len(summary.AbsorbedIntermediateRefs)
	r.promptReductionStats.recordStateSummary(absorbedCount)
	if absorbedCount == 0 {
		r.promptReductionStats.recordKeepReason("review_state_summary_current_only")
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &ReviewPromptReductionState{Mode: normalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Summary = summary
	r.promptReductionState.Report = r.promptReductionStats.reportValue()
	return text
}

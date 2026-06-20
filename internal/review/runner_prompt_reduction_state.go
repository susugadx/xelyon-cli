package review

import (
	"strings"
	"time"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func (r *ReviewRunner) probeResultPromptContextOptions() reviewmodelinput.ProbeResultPromptContextOptions {
	if r == nil {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	mode := reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == reviewpromptreduction.ReviewPromptReductionModeOff {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(mode)
	}
	return reviewmodelinput.ProbeResultPromptContextOptions{
		CommandOutputCompactor: reviewpromptreduction.NewReviewPromptCommandOutputCompactor(mode, r.promptReductionStats),
	}
}

func (r *ReviewRunner) resetPromptReductionStats() {
	if r == nil {
		return
	}
	r.promptReductionStats = reviewpromptreduction.NewStats(r.promptReductionMode)
	r.promptReductionState = &reviewpromptreduction.ReviewPromptReductionState{
		Mode:   reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode),
		Report: r.promptReductionStats.Report(),
	}
}

func normalizeReviewRunID(id string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return "review-" + time.Now().UTC().Format("20060102T150405.000000000Z")
}

func (r *ReviewRunner) recordPromptReductionItem(item reviewpromptreduction.ReviewPromptReductionItem) {
	if r == nil || reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode) == reviewpromptreduction.ReviewPromptReductionModeOff {
		return
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &reviewpromptreduction.ReviewPromptReductionState{Mode: reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Items = append(r.promptReductionState.Items, item)
	if r.promptReductionStats != nil {
		r.promptReductionStats.RecordItem(item)
		r.promptReductionState.Report = r.promptReductionStats.Report()
	}
}

// PromptReductionReport は直近 Run の review prompt 削減集計を返す。
func (r *ReviewRunner) PromptReductionReport() reviewpromptreduction.ReviewPromptReductionReport {
	if r == nil {
		return reviewpromptreduction.ReviewPromptReductionReport{}
	}
	return r.promptReductionStats.Report()
}

func (r *ReviewRunner) reviewStateSummaryPrompt(input reviewpromptreduction.ReviewStateSummaryInput) string {
	if r == nil || reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode) != reviewpromptreduction.ReviewPromptReductionModeApply {
		return ""
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(r.promptReductionMode)
	}
	summary := reviewpromptreduction.BuildReviewStateSummary(input)
	text := summary.PromptText()
	if strings.TrimSpace(text) == "" {
		return ""
	}
	absorbedCount := len(summary.AbsorbedIntermediateRefs)
	r.promptReductionStats.RecordStateSummary(absorbedCount)
	if absorbedCount == 0 {
		r.promptReductionStats.RecordKeepReason("review_state_summary_current_only")
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &reviewpromptreduction.ReviewPromptReductionState{Mode: reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Summary = summary
	r.promptReductionState.Report = r.promptReductionStats.Report()
	return text
}

func reviewPromptReductionPhase(phase ReviewModelPhase) reviewpromptreduction.ReviewModelPhase {
	return reviewpromptreduction.ReviewModelPhase(phase)
}

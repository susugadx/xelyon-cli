package probe

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateReviewProbePlanNoCandidateRiskReason(plan ReviewProbePlan, surfaceIDs map[string]struct{}) error {
	if len(plan.CandidateRisks) > 0 {
		if plan.NoCandidateRiskReason != "" {
			return fmt.Errorf("no_candidate_risk_reason must be empty when candidate_risks is non-empty")
		}
		return nil
	}
	if strings.TrimSpace(plan.NoCandidateRiskReason) == "" {
		return fmt.Errorf("no_candidate_risk_reason must be non-empty when candidate_risks is empty")
	}
	for _, surface := range plan.ImpactSurfaces {
		if _, exists := surfaceIDs[surface.ID]; exists && !reviewProbePlanReasonMentionsID(plan.NoCandidateRiskReason, surface.ID) {
			return fmt.Errorf("no_candidate_risk_reason must mention impact surface ID %q when candidate_risks is empty", surface.ID)
		}
	}
	return nil
}

func validateReviewProbePlanNoProbeCompletion(plan ReviewProbePlan, surfaceIDs, riskIDs map[string]struct{}) error {
	if strings.TrimSpace(plan.NoProbeReason) == "" {
		return fmt.Errorf("no_probe_reason must be non-empty when probes is empty")
	}
	for i, surface := range plan.ImpactSurfaces {
		field := fmt.Sprintf("impact_surfaces[%d]", i)
		if surface.Status != ReviewProbeImpactSurfaceChecked {
			return fmt.Errorf("%s.status must be %q when probes is empty: got %q", field, ReviewProbeImpactSurfaceChecked, surface.Status)
		}
		if _, exists := surfaceIDs[surface.ID]; exists && !reviewProbePlanReasonMentionsID(plan.NoProbeReason, surface.ID) {
			return fmt.Errorf("no_probe_reason must mention checked impact surface ID %q when probes is empty", surface.ID)
		}
	}
	for i, risk := range plan.CandidateRisks {
		field := fmt.Sprintf("candidate_risks[%d]", i)
		if risk.Status != ReviewProbeCandidateRiskCheckedByEvidence {
			return fmt.Errorf("%s.status must be %q when probes is empty: got %q", field, ReviewProbeCandidateRiskCheckedByEvidence, risk.Status)
		}
		if _, exists := riskIDs[risk.ID]; exists && !reviewProbePlanReasonMentionsID(plan.NoProbeReason, risk.ID) {
			return fmt.Errorf("no_probe_reason must mention checked candidate risk ID %q when probes is empty", risk.ID)
		}
	}
	return nil
}

func reviewProbePlanReasonMentionsID(reason, id string) bool {
	if id == "" {
		return false
	}

	searchFrom := 0
	for searchFrom <= len(reason) {
		relativeIndex := strings.Index(reason[searchFrom:], id)
		if relativeIndex < 0 {
			return false
		}
		start := searchFrom + relativeIndex
		end := start + len(id)
		if isReviewProbePlanReasonIDBoundaryBefore(reason, start) && isReviewProbePlanReasonIDBoundaryAfter(reason, end) {
			return true
		}
		searchFrom = start + 1
	}
	return false
}

func isReviewProbePlanReasonIDBoundaryBefore(reason string, index int) bool {
	if index <= 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(reason[:index])
	return !isReviewProbePlanReasonIDContinuationRune(r)
}

func isReviewProbePlanReasonIDBoundaryAfter(reason string, index int) bool {
	if index >= len(reason) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(reason[index:])
	return !isReviewProbePlanReasonIDContinuationRune(r)
}

func isReviewProbePlanReasonIDContinuationRune(r rune) bool {
	// reason 内の ID は文章中のトークンとして扱う。句読点の囲みは許容しつつ、
	// 英数字、ハイフン、アンダースコアは prefix 関係の別 ID として区別する。
	return isReviewProbePlanIDRune(r)
}

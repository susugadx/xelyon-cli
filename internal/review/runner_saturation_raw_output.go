package review

import (
	"fmt"
	"strings"
)

func (r *ReviewRunner) failClosedReviewSaturationByRawOutputLedger(check ReviewSaturationCheck, ledger *ReviewProbeRawOutputLedger) ReviewSaturationCheck {
	if ledger == nil || check.Status != ReviewSaturationStatusSaturated || ledger.CanAcceptSaturated {
		return check
	}
	reason := strings.TrimSpace(ledger.FailClosedReason)
	if reason == "" {
		reason = reviewProbeRawOutputReasonSaturatedRejected
	}
	if r != nil && r.promptReductionStats != nil {
		r.promptReductionStats.recordKeepReason(reviewProbeRawOutputReasonSaturatedRejected)
		r.promptReductionStats.recordKeepReason(reason)
	}
	check.Status = ReviewSaturationStatusBlocked
	check.CheckedSummary = "saturation rejected because required review probe raw output was not rehydrated: " + reason
	check.MissingSurfaceIDs = nil
	check.MissingRiskIDs = nil
	check.AdditionalFindingCandidates = nil
	check.RevisionInstructions = ""
	return check
}

func (r *ReviewRunner) failClosedReviewRevisionPromptByRawOutputLedger(check ReviewSaturationCheck, ledger *ReviewProbeRawOutputLedger) error {
	if ledger == nil || ledger.CanAcceptSaturated {
		return nil
	}
	if check.Status != ReviewSaturationStatusNeedsRevision {
		return nil
	}
	reason := strings.TrimSpace(ledger.FailClosedReason)
	if reason == "" {
		reason = reviewProbeRawOutputReasonRehydrateUnavailable
	}
	if r != nil && r.promptReductionStats != nil {
		r.promptReductionStats.recordKeepReason(reason)
	}
	return fmt.Errorf("review runner revision prompt raw output rehydrate failed closed: %s", reason)
}

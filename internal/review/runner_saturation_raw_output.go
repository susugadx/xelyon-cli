package review

import (
	"fmt"
	"strings"

	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func (r *ReviewRunner) failClosedReviewSaturationByRawOutputLedger(check reviewreport.ReviewSaturationCheck, ledger *reviewpromptreduction.ReviewProbeRawOutputLedger) reviewreport.ReviewSaturationCheck {
	if ledger == nil || check.Status != reviewreport.ReviewSaturationStatusSaturated || ledger.CanAcceptSaturated {
		return check
	}
	reason := strings.TrimSpace(ledger.FailClosedReason)
	if reason == "" {
		reason = reviewProbeRawOutputReasonSaturatedRejected
	}
	if r != nil && r.promptReductionStats != nil {
		r.promptReductionStats.RecordKeepReason(reviewProbeRawOutputReasonSaturatedRejected)
		r.promptReductionStats.RecordKeepReason(reason)
	}
	check.Status = reviewreport.ReviewSaturationStatusBlocked
	check.CheckedSummary = "saturation rejected because required review probe raw output was not rehydrated: " + reason
	check.MissingSurfaceIDs = nil
	check.MissingRiskIDs = nil
	check.AdditionalFindingCandidates = nil
	check.RevisionInstructions = ""
	return check
}

func (r *ReviewRunner) failClosedReviewRevisionPromptByRawOutputLedger(check reviewreport.ReviewSaturationCheck, ledger *reviewpromptreduction.ReviewProbeRawOutputLedger) error {
	if ledger == nil || ledger.CanAcceptSaturated {
		return nil
	}
	if check.Status != reviewreport.ReviewSaturationStatusNeedsRevision {
		return nil
	}
	reason := strings.TrimSpace(ledger.FailClosedReason)
	if reason == "" {
		reason = reviewProbeRawOutputReasonRehydrateUnavailable
	}
	if r != nil && r.promptReductionStats != nil {
		r.promptReductionStats.RecordKeepReason(reason)
	}
	return fmt.Errorf("review runner revision prompt raw output rehydrate failed closed: %s", reason)
}

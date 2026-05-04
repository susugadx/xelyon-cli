package review

import (
	"fmt"
	"strings"
)

// ValidateReviewReport は review report の verdict 契約を検証する。
func ValidateReviewReport(report ReviewReport) error {
	switch report.Verdict {
	case ReviewVerdictClean:
		if len(report.RootCauseGroups) > 0 {
			return fmt.Errorf("verdict %q requires root_cause_groups to be empty", ReviewVerdictClean)
		}
	case ReviewVerdictHasFindings:
		switch report.OverallVerificationStatus {
		case ReviewVerificationVerified, ReviewVerificationPartiallyVerified:
		default:
			return fmt.Errorf("verdict %q requires overall_verification_status to be %q or %q: got %q",
				ReviewVerdictHasFindings,
				ReviewVerificationVerified,
				ReviewVerificationPartiallyVerified,
				report.OverallVerificationStatus,
			)
		}
		if len(report.RootCauseGroups) == 0 {
			return fmt.Errorf("verdict %q requires at least one root_cause_group", ReviewVerdictHasFindings)
		}
		for i, group := range report.RootCauseGroups {
			switch group.VerificationStatus {
			case ReviewVerificationVerified, ReviewVerificationPartiallyVerified:
			default:
				return fmt.Errorf("verdict %q requires root_cause_groups[%d].verification_status to be %q or %q: got %q",
					ReviewVerdictHasFindings,
					i,
					ReviewVerificationVerified,
					ReviewVerificationPartiallyVerified,
					group.VerificationStatus,
				)
			}
		}
	case ReviewVerdictBlocked:
		if !hasBlockedReason(report) {
			return fmt.Errorf("verdict %q requires blocked reason in summary, unverified_surfaces, residual_risks, or blocked probe_summaries status", ReviewVerdictBlocked)
		}
	default:
		return fmt.Errorf("verdict must be one of %q, %q, %q: got %q", ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked, report.Verdict)
	}

	return nil
}

func hasBlockedReason(report ReviewReport) bool {
	if strings.TrimSpace(report.Summary) != "" {
		return true
	}
	if len(report.UnverifiedSurfaces) > 0 {
		return true
	}
	if len(report.ResidualRisks) > 0 {
		return true
	}
	for _, summary := range report.ProbeSummaries {
		switch summary.Status {
		case ReviewProbeBlocked, ReviewProbeTimedOut, ReviewProbeMutatedWorktree:
			return true
		}
	}
	return false
}

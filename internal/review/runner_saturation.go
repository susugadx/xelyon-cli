package review

import (
	"context"
	"fmt"
)

func (r *ReviewRunner) completeReviewReportSaturation(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, report ReviewReport, bundle ReviewEvidenceBundle, coverageAuditContext reviewCoverageAuditContext) (ReviewReport, error) {
	check, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, bundle, coverageAuditContext)
	if err != nil {
		return ReviewReport{}, err
	}

	switch check.Status {
	case ReviewSaturationStatusSaturated:
		return report, nil
	case ReviewSaturationStatusBlocked:
		return ReviewReport{}, fmt.Errorf("review runner saturation check blocked: %s", check.CheckedSummary)
	case ReviewSaturationStatusNeedsRevision:
		revisedReport, err := r.completeReviewReportRevision(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, check, bundle)
		if err != nil {
			return ReviewReport{}, err
		}
		confirmation, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, revisedReport, bundle, coverageAuditContext)
		if err != nil {
			return ReviewReport{}, err
		}
		switch confirmation.Status {
		case ReviewSaturationStatusSaturated:
			return revisedReport, nil
		case ReviewSaturationStatusBlocked:
			return ReviewReport{}, fmt.Errorf("review runner saturation check blocked after revision: %s", confirmation.CheckedSummary)
		case ReviewSaturationStatusNeedsRevision:
			return ReviewReport{}, fmt.Errorf("review runner saturation check still needs revision after one revision: %s", confirmation.RevisionInstructions)
		default:
			return ReviewReport{}, fmt.Errorf("review runner saturation check returned unknown status after revision: %q", confirmation.Status)
		}
	default:
		return ReviewReport{}, fmt.Errorf("review runner saturation check returned unknown status: %q", check.Status)
	}
}

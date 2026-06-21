package review

import (
	"context"
	"fmt"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func (r *ReviewRunner) completeReviewReportSaturation(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan reviewprobeplan.ReviewProbePlan, probeSummaries []reviewreport.ReviewProbeSummary, probeResults []reviewprobe.ReviewProbeResult, redactor reviewRunnerPromptRedactor, report reviewreport.ReviewReport, bundle reviewevidence.ReviewEvidenceBundle, coverageAuditContext reviewCoverageAuditContext) (reviewreport.ReviewReport, error) {
	check, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, bundle, coverageAuditContext)
	if err != nil {
		return reviewreport.ReviewReport{}, err
	}

	switch check.Status {
	case reviewreport.ReviewSaturationStatusSaturated:
		return report, nil
	case reviewreport.ReviewSaturationStatusBlocked:
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner saturation check blocked: %s", check.CheckedSummary)
	case reviewreport.ReviewSaturationStatusNeedsRevision:
		revisedReport, err := r.completeReviewReportRevision(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, check, bundle)
		if err != nil {
			return reviewreport.ReviewReport{}, err
		}
		confirmation, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, revisedReport, bundle, coverageAuditContext)
		if err != nil {
			return reviewreport.ReviewReport{}, err
		}
		switch confirmation.Status {
		case reviewreport.ReviewSaturationStatusSaturated:
			return revisedReport, nil
		case reviewreport.ReviewSaturationStatusBlocked:
			return reviewreport.ReviewReport{}, fmt.Errorf("review runner saturation check blocked after revision: %s", confirmation.CheckedSummary)
		case reviewreport.ReviewSaturationStatusNeedsRevision:
			return reviewreport.ReviewReport{}, fmt.Errorf("review runner saturation check still needs revision after one revision: %s", confirmation.RevisionInstructions)
		default:
			return reviewreport.ReviewReport{}, fmt.Errorf("review runner saturation check returned unknown status after revision: %q", confirmation.Status)
		}
	default:
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner saturation check returned unknown status: %q", check.Status)
	}
}

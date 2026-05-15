package review

import (
	"context"
	"fmt"
)

func (r *ReviewRunner) completeReviewReportSaturation(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, report ReviewReport) (ReviewReport, error) {
	check, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report)
	if err != nil {
		return ReviewReport{}, err
	}

	switch check.Status {
	case ReviewSaturationStatusSaturated:
		return report, nil
	case ReviewSaturationStatusBlocked:
		return ReviewReport{}, fmt.Errorf("review runner saturation check blocked: %s", check.CheckedSummary)
	case ReviewSaturationStatusNeedsRevision:
		revisedReport, err := r.completeReviewReportRevision(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, check)
		if err != nil {
			return ReviewReport{}, err
		}
		confirmation, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, revisedReport)
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

func (r *ReviewRunner) completeReviewSaturationCheck(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport) (ReviewSaturationCheck, error) {
	checkResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseSaturationCheck,
		Prompt: buildReviewSaturationCheckPrompt(req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, finalizedReport),
	})
	if err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}

	check, checkErr := finalizeReviewRunnerSaturationCheckModelOutput(checkResp.Content, plan, finalizedReport)
	if checkErr == nil {
		return check, nil
	}

	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase: ReviewModelPhaseSaturationCheck,
		Prompt: buildReviewSaturationCheckRepairPrompt(
			req,
			evidenceMarkdown,
			plan,
			probeSummaries,
			probeResults,
			redactor,
			finalizedReport,
			checkResp.Content,
			checkErr,
		),
	})
	if err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}

	return finalizeReviewRunnerSaturationCheckModelOutput(repairResp.Content, plan, finalizedReport)
}

func (r *ReviewRunner) completeReviewReportRevision(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck) (ReviewReport, error) {
	revisionResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReportRevision,
		Prompt: buildReviewReportRevisionPrompt(req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, finalizedReport, saturationCheck),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}

	report, revisionErr := finalizeReviewRunnerReportModelOutput(revisionResp.Content, plan, probeSummaries, redactor)
	if revisionErr == nil {
		return report, nil
	}

	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase: ReviewModelPhaseReportRevision,
		Prompt: buildReviewReportRevisionRepairPrompt(
			req,
			evidenceMarkdown,
			plan,
			probeSummaries,
			probeResults,
			redactor,
			finalizedReport,
			saturationCheck,
			revisionResp.Content,
			revisionErr,
		),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}

	report, err = finalizeReviewRunnerReportModelOutput(repairResp.Content, plan, probeSummaries, redactor)
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner report revision repair: %w", err)
	}

	return report, nil
}

func finalizeReviewRunnerSaturationCheckModelOutput(content string, plan ReviewProbePlan, finalizedReport ReviewReport) (ReviewSaturationCheck, error) {
	check, err := DecodeReviewSaturationCheckJSON([]byte(content), plan, finalizedReport)
	if err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner decode saturation check: %w", err)
	}
	return check, nil
}

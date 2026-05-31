package review

import (
	"context"
	"fmt"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

func (r *ReviewRunner) completeReviewReportSaturation(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, report ReviewReport, bundle ReviewEvidenceBundle) (ReviewReport, error) {
	check, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, bundle)
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
		confirmation, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, revisedReport, bundle)
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

func (r *ReviewRunner) completeReviewSaturationCheck(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, bundle ReviewEvidenceBundle) (ReviewSaturationCheck, error) {
	r.emitProgressRunning(reviewProgressSaturationCheckItem)
	checkPrompt := reviewmodelinput.BuildSaturationCheckPrompt(reviewmodelinput.SaturationCheckPromptInput{
		CustomInstructions: req.CustomInstructions,
		EvidenceMarkdown:   evidenceMarkdown,
		Plan:               plan,
		ProbeSummaries:     probeSummaries,
		ProbeResults:       probeResults,
		Redactor:           redactor,
		FinalizedReport:    finalizedReport,
	})
	r.saveReviewRunTextArtifact("saturation_prompt.md", checkPrompt, redactor)
	checkResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseSaturationCheck,
		Prompt: checkPrompt,
	})
	if err != nil {
		r.emitProgressError(reviewProgressSaturationCheckItem, err)
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}
	r.saveReviewRunTextArtifact("saturation_raw.json", checkResp.Content, redactor)

	check, checkErr := finalizeReviewRunnerSaturationCheckModelOutput(checkResp.Content, plan, finalizedReport, bundle)
	if checkErr == nil {
		r.emitProgressOK(reviewProgressSaturationCheckItem, string(check.Status))
		return check, nil
	}

	r.emitProgressRunning(reviewProgressSaturationRepairItem)
	repairPrompt := reviewmodelinput.BuildSaturationCheckRepairPrompt(reviewmodelinput.SaturationCheckRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		EvidenceMarkdown:      evidenceMarkdown,
		Plan:                  plan,
		ProbeSummaries:        probeSummaries,
		ProbeResults:          probeResults,
		Redactor:              redactor,
		FinalizedReport:       finalizedReport,
		InvalidOutput:         checkResp.Content,
		DecodeOrValidationErr: checkErr,
	})
	r.saveReviewRunTextArtifact("saturation_prompt.md", repairPrompt, redactor)
	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseSaturationCheck,
		Prompt: repairPrompt,
	})
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}
	r.saveReviewRunTextArtifact("saturation_raw.json", repairResp.Content, redactor)

	check, err = finalizeReviewRunnerSaturationCheckModelOutput(repairResp.Content, plan, finalizedReport, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, err
	}
	r.emitProgressOK(reviewProgressSaturationRepairItem, string(check.Status))
	return check, nil
}

func (r *ReviewRunner) completeReviewReportRevision(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck, bundle ReviewEvidenceBundle) (ReviewReport, error) {
	r.emitProgressRunning(reviewProgressReportRevisionItem)
	revisionPrompt := reviewmodelinput.BuildReportRevisionPrompt(reviewmodelinput.ReportRevisionPromptInput{
		CustomInstructions: req.CustomInstructions,
		EvidenceMarkdown:   evidenceMarkdown,
		Plan:               plan,
		ProbeSummaries:     probeSummaries,
		ProbeResults:       probeResults,
		Redactor:           redactor,
		FinalizedReport:    finalizedReport,
		SaturationCheck:    saturationCheck,
	})
	r.saveReviewRunTextArtifact("revision_prompt.md", revisionPrompt, redactor)
	revisionResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReportRevision,
		Prompt: revisionPrompt,
	})
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionItem, err)
		return ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}
	r.saveReviewRunTextArtifact("revision_raw.json", revisionResp.Content, redactor)

	report, revisionErr := finalizeReviewRunnerReportModelOutput(revisionResp.Content, plan, probeSummaries, redactor, bundle)
	if revisionErr == nil {
		r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
		r.emitProgressOK(reviewProgressReportRevisionItem, "")
		return report, nil
	}

	r.emitProgressRunning(reviewProgressReportRevisionRepairItem)
	repairPrompt := reviewmodelinput.BuildReportRevisionRepairPrompt(reviewmodelinput.ReportRevisionRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		EvidenceMarkdown:      evidenceMarkdown,
		Plan:                  plan,
		ProbeSummaries:        probeSummaries,
		ProbeResults:          probeResults,
		Redactor:              redactor,
		FinalizedReport:       finalizedReport,
		SaturationCheck:       saturationCheck,
		InvalidRevisionOutput: revisionResp.Content,
		DecodeOrValidationErr: revisionErr,
	})
	r.saveReviewRunTextArtifact("revision_prompt.md", repairPrompt, redactor)
	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReportRevision,
		Prompt: repairPrompt,
	})
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionRepairItem, err)
		return ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}
	r.saveReviewRunTextArtifact("revision_raw.json", repairResp.Content, redactor)

	report, err = finalizeReviewRunnerReportModelOutput(repairResp.Content, plan, probeSummaries, redactor, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionRepairItem, err)
		return ReviewReport{}, fmt.Errorf("review runner report revision repair: %w", err)
	}

	r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
	r.emitProgressOK(reviewProgressReportRevisionRepairItem, "")
	return report, nil
}

func finalizeReviewRunnerSaturationCheckModelOutput(content string, plan ReviewProbePlan, finalizedReport ReviewReport, bundle ReviewEvidenceBundle) (ReviewSaturationCheck, error) {
	check, err := DecodeReviewSaturationCheckJSON([]byte(content), plan, finalizedReport)
	if err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner decode saturation check: %w", err)
	}
	if err := validateReviewSaturationExternalDocRefsAgainstEvidence(check, bundle); err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner decode saturation check: %w", err)
	}
	return check, nil
}

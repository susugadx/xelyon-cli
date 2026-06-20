package review

import (
	"context"
	"fmt"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func (r *ReviewRunner) completeReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan reviewprobeplan.ReviewProbePlan, probeSummaries []reviewreport.ReviewProbeSummary, probeResults []reviewprobe.ReviewProbeResult, redactor reviewRunnerPromptRedactor, bundle reviewevidence.ReviewEvidenceBundle, coverageAuditContext reviewCoverageAuditContext) (reviewreport.ReviewReport, error) {
	r.emitProgressRunning(reviewProgressReportItem)
	report, err := r.completeInitialReviewReport(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressReportItem, err)
		return reviewreport.ReviewReport{}, err
	}
	r.emitProgressOK(reviewProgressReportItem, "")
	return r.completeReviewReportSaturation(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, bundle, coverageAuditContext)
}

func (r *ReviewRunner) completeInitialReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan reviewprobeplan.ReviewProbePlan, probeSummaries []reviewreport.ReviewProbeSummary, probeResults []reviewprobe.ReviewProbeResult, redactor reviewRunnerPromptRedactor, bundle reviewevidence.ReviewEvidenceBundle) (reviewreport.ReviewReport, error) {
	stateSummary := r.reviewStateSummaryPrompt(reviewpromptreduction.ReviewStateSummaryInput{
		Bundle:         bundle,
		Plan:           plan,
		ProbeSummaries: probeSummaries,
		Phase:          reviewPromptReductionPhase(ReviewModelPhaseReport),
	})
	reportPrompt := reviewmodelinput.BuildReportPrompt(reviewmodelinput.ReportPromptInput{
		CustomInstructions: req.CustomInstructions,
		ReviewStateSummary: stateSummary,
		EvidenceMarkdown:   evidenceMarkdown,
		Plan:               plan,
		ProbeSummaries:     probeSummaries,
		ProbeResults:       probeResults,
		Redactor:           redactor,
		ProbeResultOptions: r.probeResultPromptContextOptions(),
	})
	r.saveReviewRunTextArtifact("report_prompt.md", reportPrompt, redactor)
	reportResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReport,
		Prompt: reportPrompt,
	})
	if err != nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
	}
	r.saveReviewRunTextArtifact("report_raw.json", reportResp.Content, redactor)

	report, reportErr := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               reportResp.Content,
		Plan:                  plan,
		TrustedProbeSummaries: probeSummaries,
		Redactor:              redactor,
		ExternalDocs:          bundle.WebSearchEvidence.ExternalDocs,
	})
	if reportErr == nil {
		r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
		return report, nil
	}

	repairPrompt := reviewmodelinput.BuildReportRepairPrompt(reviewmodelinput.ReportRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		ReviewStateSummary:    stateSummary,
		EvidenceMarkdown:      evidenceMarkdown,
		Plan:                  plan,
		ProbeSummaries:        probeSummaries,
		ProbeResults:          probeResults,
		Redactor:              redactor,
		ProbeResultOptions:    r.probeResultPromptContextOptions(),
		InvalidOutput:         reportResp.Content,
		DecodeOrValidationErr: reportErr,
	})
	r.saveReviewRunTextArtifact("report_prompt.md", repairPrompt, redactor)
	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReport,
		Prompt: repairPrompt,
	})
	if err != nil {
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
	}
	r.saveReviewRunTextArtifact("report_raw.json", repairResp.Content, redactor)

	report, err = reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               repairResp.Content,
		Plan:                  plan,
		TrustedProbeSummaries: probeSummaries,
		Redactor:              redactor,
		ExternalDocs:          bundle.WebSearchEvidence.ExternalDocs,
	})
	if err != nil {
		return reviewreport.ReviewReport{}, err
	}
	r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
	return report, nil
}

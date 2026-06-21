package review

import (
	"context"
	"fmt"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewReportRevisionInput struct {
	req              ReviewRequest
	evidenceMarkdown string
	plan             reviewprobeplan.ReviewProbePlan
	probeSummaries   []reviewreport.ReviewProbeSummary
	probeResults     []reviewprobe.ReviewProbeResult
	redactor         reviewRunnerPromptRedactor
	finalizedReport  reviewreport.ReviewReport
	saturationCheck  reviewreport.ReviewSaturationCheck
	bundle           reviewevidence.ReviewEvidenceBundle
}

type reviewReportRevisionPromptContext struct {
	stateSummary          string
	phaseEvidenceMarkdown string
	probePrompt           reviewProbeResultPromptContextBuild
	externalDocs          []externaldoc.Evidence
}

func (r *ReviewRunner) completeReviewReportRevision(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan reviewprobeplan.ReviewProbePlan, probeSummaries []reviewreport.ReviewProbeSummary, probeResults []reviewprobe.ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport reviewreport.ReviewReport, saturationCheck reviewreport.ReviewSaturationCheck, bundle reviewevidence.ReviewEvidenceBundle) (reviewreport.ReviewReport, error) {
	input := reviewReportRevisionInput{
		req:              req,
		evidenceMarkdown: evidenceMarkdown,
		plan:             plan,
		probeSummaries:   probeSummaries,
		probeResults:     probeResults,
		redactor:         redactor,
		finalizedReport:  finalizedReport,
		saturationCheck:  saturationCheck,
		bundle:           bundle,
	}

	r.emitProgressRunning(reviewProgressReportRevisionItem)
	promptContext := r.buildReviewReportRevisionPromptContext(ctx, input)
	if err := r.failClosedReviewRevisionPromptByRawOutputLedger(input.saturationCheck, promptContext.probePrompt.rawOutputLedger); err != nil {
		r.emitProgressError(reviewProgressReportRevisionItem, err)
		return reviewreport.ReviewReport{}, err
	}

	revisionResp, err := r.requestReviewReportRevision(ctx, input, promptContext)
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionItem, err)
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}

	report, revisionErr := finalizeReviewReportRevisionModelOutput(revisionResp.Content, input, promptContext)
	if revisionErr == nil {
		r.saveReviewRunJSONArtifact("report_final.json", report, input.redactor)
		r.emitProgressOK(reviewProgressReportRevisionItem, "")
		return report, nil
	}

	r.emitProgressRunning(reviewProgressReportRevisionRepairItem)
	repairResp, err := r.requestReviewReportRevisionRepair(ctx, input, promptContext, revisionResp.Content, revisionErr)
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionRepairItem, err)
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner report revision model: %w", err)
	}

	report, err = finalizeReviewReportRevisionModelOutput(repairResp.Content, input, promptContext)
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionRepairItem, err)
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner report revision repair: %w", err)
	}

	r.saveReviewRunJSONArtifact("report_final.json", report, input.redactor)
	r.emitProgressOK(reviewProgressReportRevisionRepairItem, "")
	return report, nil
}

func (r *ReviewRunner) buildReviewReportRevisionPromptContext(ctx context.Context, input reviewReportRevisionInput) reviewReportRevisionPromptContext {
	return reviewReportRevisionPromptContext{
		stateSummary: r.reviewStateSummaryPrompt(reviewpromptreduction.ReviewStateSummaryInput{
			Bundle:          input.bundle,
			Plan:            input.plan,
			ProbeSummaries:  input.probeSummaries,
			FinalizedReport: input.finalizedReport,
			SaturationCheck: input.saturationCheck,
			Phase:           reviewPromptReductionPhase(ReviewModelPhaseReportRevision),
		}),
		phaseEvidenceMarkdown: r.reviewPromptEvidenceMarkdownForAbsorbedReport(ReviewModelPhaseReportRevision, input.bundle, input.evidenceMarkdown, input.finalizedReport),
		probePrompt:           r.probeResultPromptContextBuildForAbsorbedReport(ctx, ReviewModelPhaseReportRevision, "report_revision", input.finalizedReport, input.probeResults, input.redactor),
		externalDocs:          input.bundle.WebSearchEvidence.ExternalDocs,
	}
}

func (r *ReviewRunner) requestReviewReportRevision(ctx context.Context, input reviewReportRevisionInput, promptContext reviewReportRevisionPromptContext) (ReviewModelResponse, error) {
	revisionPrompt := reviewmodelinput.BuildReportRevisionPrompt(reviewmodelinput.ReportRevisionPromptInput{
		CustomInstructions:          input.req.CustomInstructions,
		ReviewStateSummary:          promptContext.stateSummary,
		EvidenceMarkdown:            promptContext.phaseEvidenceMarkdown,
		Plan:                        input.plan,
		ProbeSummaries:              input.probeSummaries,
		ProbeResults:                input.probeResults,
		Redactor:                    input.redactor,
		ProbeResultOptions:          promptContext.probePrompt.options,
		ReviewProbeRawOutputContext: promptContext.probePrompt.rawOutputContext,
		ReviewProbeRawOutputLedger:  promptContext.probePrompt.rawOutputLedger,
		FinalizedReport:             input.finalizedReport,
		SaturationCheck:             input.saturationCheck,
	})
	r.saveReviewRunTextArtifact("revision_prompt.md", revisionPrompt, input.redactor)
	resp, err := r.model.CompleteReview(ctx, newReviewModelRequest(ReviewModelPhaseReportRevision, revisionPrompt))
	if err != nil {
		return ReviewModelResponse{}, err
	}
	r.saveReviewRunTextArtifact("revision_raw.json", resp.Content, input.redactor)
	return resp, nil
}

func (r *ReviewRunner) requestReviewReportRevisionRepair(ctx context.Context, input reviewReportRevisionInput, promptContext reviewReportRevisionPromptContext, invalidRevisionOutput string, decodeOrValidationErr error) (ReviewModelResponse, error) {
	repairPrompt := reviewmodelinput.BuildReportRevisionRepairPrompt(reviewmodelinput.ReportRevisionRepairPromptInput{
		CustomInstructions:          input.req.CustomInstructions,
		ReviewStateSummary:          promptContext.stateSummary,
		EvidenceMarkdown:            promptContext.phaseEvidenceMarkdown,
		Plan:                        input.plan,
		ProbeSummaries:              input.probeSummaries,
		ProbeResults:                input.probeResults,
		Redactor:                    input.redactor,
		ProbeResultOptions:          promptContext.probePrompt.options,
		ReviewProbeRawOutputContext: promptContext.probePrompt.rawOutputContext,
		ReviewProbeRawOutputLedger:  promptContext.probePrompt.rawOutputLedger,
		FinalizedReport:             input.finalizedReport,
		SaturationCheck:             input.saturationCheck,
		InvalidRevisionOutput:       invalidRevisionOutput,
		DecodeOrValidationErr:       decodeOrValidationErr,
	})
	r.saveReviewRunTextArtifact("revision_prompt.md", repairPrompt, input.redactor)
	resp, err := r.model.CompleteReview(ctx, newReviewModelRequest(ReviewModelPhaseReportRevision, repairPrompt))
	if err != nil {
		return ReviewModelResponse{}, err
	}
	r.saveReviewRunTextArtifact("revision_raw.json", resp.Content, input.redactor)
	return resp, nil
}

func finalizeReviewReportRevisionModelOutput(content string, input reviewReportRevisionInput, promptContext reviewReportRevisionPromptContext) (reviewreport.ReviewReport, error) {
	return reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               content,
		Plan:                  input.plan,
		TrustedProbeSummaries: input.probeSummaries,
		Redactor:              input.redactor,
		ExternalDocs:          promptContext.externalDocs,
	})
}

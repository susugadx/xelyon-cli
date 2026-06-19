package review

import (
	"context"
	"fmt"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
)

type reviewSaturationCheckInput struct {
	req                  ReviewRequest
	evidenceMarkdown     string
	plan                 ReviewProbePlan
	probeSummaries       []ReviewProbeSummary
	probeResults         []ReviewProbeResult
	redactor             reviewRunnerPromptRedactor
	finalizedReport      ReviewReport
	bundle               ReviewEvidenceBundle
	coverageAuditContext reviewCoverageAuditContext
}

type reviewSaturationCheckPromptContext struct {
	stateSummary          string
	phaseEvidenceMarkdown string
	probePrompt           reviewProbeResultPromptContextBuild
	externalDocs          []ReviewExternalDocEvidence
}

func (r *ReviewRunner) completeReviewSaturationCheck(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, bundle ReviewEvidenceBundle, coverageAuditContext reviewCoverageAuditContext) (ReviewSaturationCheck, error) {
	input := reviewSaturationCheckInput{
		req:                  req,
		evidenceMarkdown:     evidenceMarkdown,
		plan:                 plan,
		probeSummaries:       probeSummaries,
		probeResults:         probeResults,
		redactor:             redactor,
		finalizedReport:      finalizedReport,
		bundle:               bundle,
		coverageAuditContext: coverageAuditContext,
	}

	r.emitProgressRunning(reviewProgressSaturationCheckItem)
	promptContext := r.buildReviewSaturationCheckPromptContext(ctx, input)
	checkResp, err := r.requestReviewSaturationCheck(ctx, input, promptContext)
	if err != nil {
		r.emitProgressError(reviewProgressSaturationCheckItem, err)
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}

	check, checkErr := finalizeReviewSaturationCheckModelOutput(checkResp.Content, input, promptContext)
	if checkErr == nil {
		return r.acceptReviewSaturationCheck(input, promptContext, reviewProgressSaturationCheckItem, check)
	}

	r.emitProgressRunning(reviewProgressSaturationRepairItem)
	repairResp, err := r.requestReviewSaturationCheckRepair(ctx, input, promptContext, checkResp.Content, checkErr)
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, fmt.Errorf("review runner saturation check model: %w", err)
	}

	check, err = finalizeReviewSaturationCheckModelOutput(repairResp.Content, input, promptContext)
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, err
	}
	return r.acceptReviewSaturationCheck(input, promptContext, reviewProgressSaturationRepairItem, check)
}

func (r *ReviewRunner) buildReviewSaturationCheckPromptContext(ctx context.Context, input reviewSaturationCheckInput) reviewSaturationCheckPromptContext {
	return reviewSaturationCheckPromptContext{
		stateSummary: r.reviewStateSummaryPrompt(reviewStateSummaryInput{
			bundle:          input.bundle,
			plan:            input.plan,
			probeSummaries:  input.probeSummaries,
			finalizedReport: input.finalizedReport,
			phase:           ReviewModelPhaseSaturationCheck,
		}),
		phaseEvidenceMarkdown: r.reviewPromptEvidenceMarkdownForAbsorbedReport(ReviewModelPhaseSaturationCheck, input.bundle, input.evidenceMarkdown, input.finalizedReport),
		probePrompt:           r.probeResultPromptContextBuildForAbsorbedReport(ctx, ReviewModelPhaseSaturationCheck, "saturation_check", input.finalizedReport, input.probeResults, input.redactor),
		externalDocs:          input.bundle.WebSearchEvidence.ExternalDocs,
	}
}

func (r *ReviewRunner) requestReviewSaturationCheck(ctx context.Context, input reviewSaturationCheckInput, promptContext reviewSaturationCheckPromptContext) (ReviewModelResponse, error) {
	checkPrompt := reviewmodelinput.BuildSaturationCheckPrompt(reviewmodelinput.SaturationCheckPromptInput{
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
	})
	r.saveReviewRunTextArtifact("saturation_prompt.md", checkPrompt, input.redactor)
	resp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseSaturationCheck,
		Prompt: checkPrompt,
	})
	if err != nil {
		return ReviewModelResponse{}, err
	}
	r.saveReviewRunTextArtifact("saturation_raw.json", resp.Content, input.redactor)
	return resp, nil
}

func (r *ReviewRunner) requestReviewSaturationCheckRepair(ctx context.Context, input reviewSaturationCheckInput, promptContext reviewSaturationCheckPromptContext, invalidOutput string, decodeOrValidationErr error) (ReviewModelResponse, error) {
	repairPrompt := reviewmodelinput.BuildSaturationCheckRepairPrompt(reviewmodelinput.SaturationCheckRepairPromptInput{
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
		InvalidOutput:               invalidOutput,
		DecodeOrValidationErr:       decodeOrValidationErr,
	})
	r.saveReviewRunTextArtifact("saturation_prompt.md", repairPrompt, input.redactor)
	resp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseSaturationCheck,
		Prompt: repairPrompt,
	})
	if err != nil {
		return ReviewModelResponse{}, err
	}
	r.saveReviewRunTextArtifact("saturation_raw.json", resp.Content, input.redactor)
	return resp, nil
}

func finalizeReviewSaturationCheckModelOutput(content string, input reviewSaturationCheckInput, promptContext reviewSaturationCheckPromptContext) (ReviewSaturationCheck, error) {
	return reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         content,
		Plan:            input.plan,
		FinalizedReport: input.finalizedReport,
		ExternalDocs:    promptContext.externalDocs,
	})
}

func (r *ReviewRunner) acceptReviewSaturationCheck(input reviewSaturationCheckInput, promptContext reviewSaturationCheckPromptContext, progressItem reviewProgressItem, check ReviewSaturationCheck) (ReviewSaturationCheck, error) {
	merged, err := mergeReviewCoverageAuditIntoSaturationCheck(check, input.plan, input.finalizedReport, input.probeSummaries, input.coverageAuditContext)
	if err != nil {
		r.emitProgressError(progressItem, err)
		return ReviewSaturationCheck{}, err
	}
	merged = r.failClosedReviewSaturationByRawOutputLedger(merged, promptContext.probePrompt.rawOutputLedger)
	r.emitProgressOK(progressItem, string(merged.Status))
	return merged, nil
}

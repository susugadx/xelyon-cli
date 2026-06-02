package review

import (
	"context"
	"fmt"
	"strings"

	reviewanalysis "github.com/susugadx/xelyon-cli/internal/review/analysis"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

type reviewCoverageAuditContext struct {
	postPass1ExternalEvidence reviewreport.CoverageExternalEvidenceDelta
	externalSupport           reviewreport.CoverageExternalSupport
}

func (r *ReviewRunner) completeReviewReportSaturation(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, report ReviewReport, externalDocs []ReviewExternalDocEvidence, coverageAuditContext reviewCoverageAuditContext) (ReviewReport, error) {
	check, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, externalDocs, coverageAuditContext)
	if err != nil {
		return ReviewReport{}, err
	}

	switch check.Status {
	case ReviewSaturationStatusSaturated:
		return report, nil
	case ReviewSaturationStatusBlocked:
		return ReviewReport{}, fmt.Errorf("review runner saturation check blocked: %s", check.CheckedSummary)
	case ReviewSaturationStatusNeedsRevision:
		revisedReport, err := r.completeReviewReportRevision(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, check, externalDocs)
		if err != nil {
			return ReviewReport{}, err
		}
		confirmation, err := r.completeReviewSaturationCheck(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, revisedReport, externalDocs, coverageAuditContext)
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

func (r *ReviewRunner) completeReviewSaturationCheck(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, externalDocs []ReviewExternalDocEvidence, coverageAuditContext reviewCoverageAuditContext) (ReviewSaturationCheck, error) {
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

	check, checkErr := reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         checkResp.Content,
		Plan:            plan,
		FinalizedReport: finalizedReport,
		ExternalDocs:    externalDocs,
	})
	if checkErr == nil {
		check, err = mergeReviewCoverageAuditIntoSaturationCheck(check, plan, finalizedReport, probeSummaries, coverageAuditContext)
		if err != nil {
			r.emitProgressError(reviewProgressSaturationCheckItem, err)
			return ReviewSaturationCheck{}, err
		}
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

	check, err = reviewmodeloutput.FinalizeSaturationCheckModelOutput(reviewmodeloutput.SaturationCheckModelOutputInput{
		Content:         repairResp.Content,
		Plan:            plan,
		FinalizedReport: finalizedReport,
		ExternalDocs:    externalDocs,
	})
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, err
	}
	check, err = mergeReviewCoverageAuditIntoSaturationCheck(check, plan, finalizedReport, probeSummaries, coverageAuditContext)
	if err != nil {
		r.emitProgressError(reviewProgressSaturationRepairItem, err)
		return ReviewSaturationCheck{}, err
	}
	r.emitProgressOK(reviewProgressSaturationRepairItem, string(check.Status))
	return check, nil
}

func buildReviewCoverageAuditContext(before ReviewWebSearchEvidence, bundle ReviewEvidenceBundle) reviewCoverageAuditContext {
	support := BuildReviewEvidenceModelInput(bundle).ExternalSupport
	externalSupport := reviewreport.CoverageExternalSupport{
		Level:                                string(support.Level),
		DocCount:                             support.DocCount,
		CitationCapableDocCount:              support.CitationCapableDocCount,
		CitationCapableSnippetCount:          support.CitationCapableSnippetCount,
		OfficialCandidateDocCount:            support.OfficialCandidateDocCount,
		OfficialCandidateCitationCapableDocs: support.OfficialCandidateCitationCapableDocCount,
		OfficialConfirmation:                 support.OfficialConfirmation,
		Warnings:                             append([]string(nil), support.Warnings...),
		Reasons:                              append([]string(nil), support.Reasons...),
	}
	return reviewCoverageAuditContext{
		postPass1ExternalEvidence: buildReviewCoverageExternalEvidenceDelta(before, bundle.WebSearchEvidence, externalSupport),
		externalSupport:           externalSupport,
	}
}

func buildReviewCoverageExternalEvidenceDelta(before, after ReviewWebSearchEvidence, support reviewreport.CoverageExternalSupport) reviewreport.CoverageExternalEvidenceDelta {
	delta := reviewreport.CoverageExternalEvidenceDelta{}

	addedQueries := addedReviewWebSearchEvidenceQueries(before, after)
	delta.AddedQueryCount = len(addedQueries)
	for _, query := range addedQueries {
		queryText := strings.TrimSpace(query.Query)
		if queryText != "" {
			delta.AddedQueries = append(delta.AddedQueries, queryText)
		}
		if strings.TrimSpace(query.Error) != "" {
			delta.AddedFailedQueryCount++
			if queryText != "" {
				delta.AddedFailedQueries = append(delta.AddedFailedQueries, queryText)
			}
		}
		if len(query.Results) == 0 {
			delta.AddedNoResultCount++
			if queryText != "" {
				delta.AddedNoResultQueries = append(delta.AddedNoResultQueries, queryText)
			}
		}
	}

	for _, doc := range addedReviewExternalDocs(before, after) {
		if docID := strings.TrimSpace(doc.DocID); docID != "" {
			delta.AddedDocIDs = append(delta.AddedDocIDs, docID)
		}
		if docURL := strings.TrimSpace(doc.URL); docURL != "" {
			delta.AddedDocURLs = append(delta.AddedDocURLs, docURL)
		}
		if strings.TrimSpace(doc.Error) != "" {
			delta.EvidenceError = true
		}
		if doc.Truncated {
			delta.Truncated = true
		}
	}
	if after.Truncated && !before.Truncated {
		delta.Truncated = true
	}
	if after.Inconclusive && (!before.Inconclusive || delta.AddedQueryCount > 0 || len(delta.AddedDocIDs) > 0 || len(delta.AddedDocURLs) > 0) {
		delta.Inconclusive = true
	}
	if strings.TrimSpace(after.Error) != "" && strings.TrimSpace(after.Error) != strings.TrimSpace(before.Error) {
		delta.EvidenceError = true
	}
	if delta.AddedQueryCount == 0 &&
		len(delta.AddedDocIDs) == 0 &&
		len(delta.AddedDocURLs) == 0 &&
		!delta.EvidenceError &&
		!delta.Inconclusive &&
		!delta.Truncated {
		return reviewreport.CoverageExternalEvidenceDelta{}
	}
	delta.Warnings = append([]string(nil), support.Warnings...)
	delta.Reasons = append([]string(nil), support.Reasons...)
	return delta
}

func addedReviewWebSearchEvidenceQueries(before, after ReviewWebSearchEvidence) []ReviewWebSearchEvidenceQuery {
	if len(after.Queries) <= len(before.Queries) {
		return nil
	}
	return append([]ReviewWebSearchEvidenceQuery(nil), after.Queries[len(before.Queries):]...)
}

func addedReviewExternalDocs(before, after ReviewWebSearchEvidence) []ReviewExternalDocEvidence {
	if len(after.ExternalDocs) <= len(before.ExternalDocs) {
		return nil
	}
	return append([]ReviewExternalDocEvidence(nil), after.ExternalDocs[len(before.ExternalDocs):]...)
}

func mergeReviewCoverageAuditIntoSaturationCheck(check ReviewSaturationCheck, plan ReviewProbePlan, finalizedReport ReviewReport, probeSummaries []ReviewProbeSummary, auditContext reviewCoverageAuditContext) (ReviewSaturationCheck, error) {
	planScope := reviewanalysis.PlanScopeFromProbePlan(plan)
	issues := reviewreport.AuditReviewReportCoverage(reviewreport.CoverageAuditInput{
		Plan:                      planScope,
		Report:                    finalizedReport,
		TrustedProbeSummaries:     probeSummaries,
		PostPass1ExternalEvidence: auditContext.postPass1ExternalEvidence,
		ExternalSupport:           auditContext.externalSupport,
	})
	merged := reviewreport.MergeCoverageIssuesIntoSaturationCheck(check, issues)
	if err := reviewreport.ValidateReviewSaturationCheck(merged, planScope, finalizedReport); err != nil {
		return ReviewSaturationCheck{}, fmt.Errorf("review runner merge coverage audit: %w", err)
	}
	return merged, nil
}

func (r *ReviewRunner) completeReviewReportRevision(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, finalizedReport ReviewReport, saturationCheck ReviewSaturationCheck, externalDocs []ReviewExternalDocEvidence) (ReviewReport, error) {
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

	report, revisionErr := reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               revisionResp.Content,
		Plan:                  plan,
		TrustedProbeSummaries: probeSummaries,
		Redactor:              redactor,
		ExternalDocs:          externalDocs,
	})
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

	report, err = reviewmodeloutput.FinalizeReportModelOutput(reviewmodeloutput.ReportModelOutputInput{
		Content:               repairResp.Content,
		Plan:                  plan,
		TrustedProbeSummaries: probeSummaries,
		Redactor:              redactor,
		ExternalDocs:          externalDocs,
	})
	if err != nil {
		r.emitProgressError(reviewProgressReportRevisionRepairItem, err)
		return ReviewReport{}, fmt.Errorf("review runner report revision repair: %w", err)
	}

	r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
	r.emitProgressOK(reviewProgressReportRevisionRepairItem, "")
	return report, nil
}

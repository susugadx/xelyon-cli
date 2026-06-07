package review

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
)

var (
	errReviewRunnerModelNil           = errors.New("review runner model is nil")
	errReviewRunnerEvidenceBuilderNil = errors.New("review runner evidence builder is nil")
	errReviewRunnerProbeRunnerNil     = errors.New("review runner probe runner is nil")
)

// ReviewEvidenceProvider は ReviewRunner が使う evidence 収集境界を表す。
type ReviewEvidenceProvider interface {
	BuildCurrentChanges(context.Context) (ReviewEvidenceBundle, error)
}

// ReviewProbeExecutor は ReviewRunner が使う probe 実行境界を表す。
type ReviewProbeExecutor interface {
	Run(context.Context, ReviewProbeRequest) (ReviewProbeResult, error)
}

// ReviewRunnerOptions は ReviewRunner の必須依存関係を表す。
// concrete 依存の選択と構築は adapter/factory 側の責務にし、
// ReviewRunner は注入済みの review domain 境界だけを使う。
type ReviewRunnerOptions struct {
	EvidenceBuilder ReviewEvidenceProvider
	ProbeRunner     ReviewProbeExecutor
	Model           ReviewModel

	ArtifactWriter        ReviewRunArtifactWriter
	ArtifactWarningWriter io.Writer
	ProgressSink          ReviewProgressSink
	PromptReductionMode   ReviewPromptReductionMode

	RawOutputArtifactsMode            ReviewRawOutputArtifactsMode
	RawOutputArtifactStore            ReviewRawOutputArtifactStore
	RawOutputSessionID                string
	ReviewRunID                       string
	RawOutputRehydrateBudgetTokens    int
	RawOutputRehydrateBudgetMaxTokens int
}

// ReviewRunner は /review current_changes の evidence、model、probe、report を順に束ねる。
type ReviewRunner struct {
	evidenceBuilder ReviewEvidenceProvider
	probeRunner     ReviewProbeExecutor
	model           ReviewModel

	artifactWriter                    ReviewRunArtifactWriter
	artifactWarningWriter             io.Writer
	progressSink                      ReviewProgressSink
	promptReductionMode               ReviewPromptReductionMode
	rawOutputArtifactsMode            ReviewRawOutputArtifactsMode
	rawOutputArtifactStore            ReviewRawOutputArtifactStore
	rawOutputSessionID                string
	reviewRunID                       string
	rawOutputRehydrateBudgetTokens    int
	rawOutputRehydrateBudgetMaxTokens int
	promptReductionStats              *reviewPromptReductionStats
	promptReductionState              *ReviewPromptReductionState
}

// NewReviewRunner は ReviewRunner を構築し、必須依存を検証する。
func NewReviewRunner(opts ReviewRunnerOptions) (*ReviewRunner, error) {
	if err := validateReviewRunnerOptions(opts); err != nil {
		return nil, err
	}
	return &ReviewRunner{
		evidenceBuilder:                   opts.EvidenceBuilder,
		probeRunner:                       opts.ProbeRunner,
		model:                             opts.Model,
		artifactWriter:                    opts.ArtifactWriter,
		artifactWarningWriter:             opts.ArtifactWarningWriter,
		progressSink:                      opts.ProgressSink,
		promptReductionMode:               normalizeReviewPromptReductionMode(opts.PromptReductionMode),
		rawOutputArtifactsMode:            normalizeReviewRawOutputArtifactsMode(opts.RawOutputArtifactsMode),
		rawOutputArtifactStore:            opts.RawOutputArtifactStore,
		rawOutputSessionID:                strings.TrimSpace(opts.RawOutputSessionID),
		reviewRunID:                       normalizeReviewRunID(opts.ReviewRunID),
		rawOutputRehydrateBudgetTokens:    opts.RawOutputRehydrateBudgetTokens,
		rawOutputRehydrateBudgetMaxTokens: opts.RawOutputRehydrateBudgetMaxTokens,
	}, nil
}

// Run は current_changes review の MVP orchestration を実行する。
func (r *ReviewRunner) Run(ctx context.Context, req ReviewRequest) (ReviewReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.validate(); err != nil {
		return ReviewReport{}, err
	}
	r.resetPromptReductionStats()
	if req.TargetKind != TargetCurrentChanges {
		return ReviewReport{}, fmt.Errorf("review runner target_kind must be %q: got %q", TargetCurrentChanges, req.TargetKind)
	}

	r.emitProgressRunning(reviewProgressEvidenceItem)
	bundle, err := r.evidenceBuilder.BuildCurrentChanges(ctx)
	if err != nil {
		r.emitProgressError(reviewProgressEvidenceItem, err)
		return ReviewReport{}, fmt.Errorf("review runner build evidence: %w", err)
	}
	r.emitProgressOK(reviewProgressEvidenceItem, reviewEvidenceProgressDetail(bundle))
	evidenceMarkdown := RenderReviewEvidenceMarkdown(bundle)
	evidenceRedactor := newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence.md", evidenceMarkdown, evidenceRedactor)
	if bundle.WebSearchEvidence.Enabled {
		r.saveReviewRunJSONArtifact("web_search_evidence.json", bundle.WebSearchEvidence, evidenceRedactor)
	}

	r.emitProgressRunning(reviewProgressProbePlanItem)
	plan, err := r.completeReviewProbePlan(ctx, req, evidenceMarkdown, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressProbePlanItem, err)
		return ReviewReport{}, err
	}
	bundle, evidenceMarkdown, evidenceRedactor, coverageAuditContext := r.collectPostPass1WebSearchEvidence(ctx, bundle, plan, evidenceMarkdown, evidenceRedactor)
	probeRequests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		r.emitProgressError(reviewProgressProbePlanItem, err)
		return ReviewReport{}, fmt.Errorf("review runner build probe requests: %w", err)
	}
	r.emitProgressOK(reviewProgressProbePlanItem, reviewProgressProbeCountDetail(len(probeRequests)))
	r.saveReviewRunJSONArtifact("probe_requests.json", probeRequests, evidenceRedactor)

	probeResults, err := r.runReviewProbesSequentially(ctx, probeRequests)
	if err != nil {
		return ReviewReport{}, err
	}
	probeSummaries := BuildReviewProbeSummaries(probeResults)
	redactor := newReviewRunnerPromptRedactor(bundle, probeResults)
	r.saveReviewRunJSONArtifact("probe_results.json", reviewmodelinput.BuildProbeResultPromptContexts(probeResults, redactor), redactor)

	reportEvidenceMarkdown := r.reviewPromptEvidenceMarkdown(bundle, evidenceMarkdown)
	return r.completeReviewReport(ctx, req, reportEvidenceMarkdown, plan, probeSummaries, probeResults, redactor, bundle, coverageAuditContext)
}

func (r *ReviewRunner) collectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan ReviewProbePlan, evidenceMarkdown string, redactor reviewRunnerPromptRedactor) (ReviewEvidenceBundle, string, reviewRunnerPromptRedactor, reviewCoverageAuditContext) {
	if !bundle.WebSearchEvidence.Enabled {
		return bundle, evidenceMarkdown, redactor, reviewCoverageAuditContext{}
	}
	provider, ok := r.evidenceBuilder.(ReviewPostPass1WebSearchEvidenceProvider)
	if !ok {
		return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(bundle.WebSearchEvidence, bundle)
	}
	before := bundle.WebSearchEvidence
	bundle.WebSearchEvidence = provider.CollectPostPass1WebSearchEvidence(ctx, bundle, plan)
	evidenceMarkdown = RenderReviewEvidenceMarkdown(bundle)
	redactor = newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence_post_pass1.md", evidenceMarkdown, redactor)
	r.saveReviewRunJSONArtifact("web_search_evidence_post_pass1.json", bundle.WebSearchEvidence, redactor)
	return bundle, evidenceMarkdown, redactor, buildReviewCoverageAuditContext(before, bundle)
}

func (r *ReviewRunner) completeReviewProbePlan(ctx context.Context, req ReviewRequest, evidenceMarkdown string, bundle ReviewEvidenceBundle) (ReviewProbePlan, error) {
	planPrompt := reviewmodelinput.BuildProbePlanPrompt(reviewmodelinput.ProbePlanPromptInput{
		CustomInstructions: req.CustomInstructions,
		EvidenceMarkdown:   evidenceMarkdown,
	})
	redactor := newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("probe_plan_prompt.md", planPrompt, redactor)
	planResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: planPrompt,
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}
	r.saveReviewRunTextArtifact("probe_plan_raw.json", planResp.Content, redactor)

	plan, decodeErr := decodeReviewProbePlanJSONAgainstEvidence(planResp.Content, bundle)
	if decodeErr == nil {
		r.saveReviewRunJSONArtifact("probe_plan_final.json", plan, redactor)
		return plan, nil
	}

	repairPrompt := reviewmodelinput.BuildProbePlanRepairPrompt(reviewmodelinput.ProbePlanRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		EvidenceMarkdown:      evidenceMarkdown,
		InvalidOutput:         planResp.Content,
		DecodeOrValidationErr: decodeErr,
	})
	r.saveReviewRunTextArtifact("probe_plan_prompt.md", repairPrompt, redactor)
	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: repairPrompt,
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}
	r.saveReviewRunTextArtifact("probe_plan_raw.json", repairResp.Content, redactor)

	plan, decodeErr = decodeReviewProbePlanJSONAgainstEvidence(repairResp.Content, bundle)
	if decodeErr != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner decode probe plan: %w", decodeErr)
	}
	r.saveReviewRunJSONArtifact("probe_plan_final.json", plan, redactor)
	return plan, nil
}

func decodeReviewProbePlanJSONAgainstEvidence(content string, bundle ReviewEvidenceBundle) (ReviewProbePlan, error) {
	plan, err := DecodeReviewProbePlanJSON([]byte(content))
	if err != nil {
		return ReviewProbePlan{}, err
	}
	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		return ReviewProbePlan{}, fmt.Errorf("ValidateReviewProbePlanAgainstEvidence: %w", err)
	}
	return plan, nil
}

func (r *ReviewRunner) completeReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, bundle ReviewEvidenceBundle, coverageAuditContext reviewCoverageAuditContext) (ReviewReport, error) {
	r.emitProgressRunning(reviewProgressReportItem)
	report, err := r.completeInitialReviewReport(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressReportItem, err)
		return ReviewReport{}, err
	}
	r.emitProgressOK(reviewProgressReportItem, "")
	return r.completeReviewReportSaturation(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, bundle, coverageAuditContext)
}

func (r *ReviewRunner) completeInitialReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, bundle ReviewEvidenceBundle) (ReviewReport, error) {
	stateSummary := r.reviewStateSummaryPrompt(reviewStateSummaryInput{
		bundle:         bundle,
		plan:           plan,
		probeSummaries: probeSummaries,
		phase:          ReviewModelPhaseReport,
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
		return ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
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
		return ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
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
		return ReviewReport{}, err
	}
	r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
	return report, nil
}

func (r *ReviewRunner) probeResultPromptContextOptions() reviewmodelinput.ProbeResultPromptContextOptions {
	if r == nil {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	mode := normalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == ReviewPromptReductionModeOff {
		return reviewmodelinput.ProbeResultPromptContextOptions{}
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(mode)
	}
	return reviewmodelinput.ProbeResultPromptContextOptions{
		CommandOutputCompactor: newReviewPromptCommandOutputCompactor(mode, r.promptReductionStats),
	}
}

func (r *ReviewRunner) resetPromptReductionStats() {
	if r == nil {
		return
	}
	r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	r.promptReductionState = &ReviewPromptReductionState{
		Mode:   normalizeReviewPromptReductionMode(r.promptReductionMode),
		Report: r.promptReductionStats.reportValue(),
	}
}

func normalizeReviewRunID(id string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return "review-" + time.Now().UTC().Format("20060102T150405.000000000Z")
}

func (r *ReviewRunner) recordPromptReductionItem(item ReviewPromptReductionItem) {
	if r == nil || normalizeReviewPromptReductionMode(r.promptReductionMode) == ReviewPromptReductionModeOff {
		return
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &ReviewPromptReductionState{Mode: normalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Items = append(r.promptReductionState.Items, item)
	if r.promptReductionStats != nil {
		r.promptReductionStats.recordItem(item)
		r.promptReductionState.Report = r.promptReductionStats.reportValue()
	}
}

// PromptReductionReport は直近 Run の review prompt 削減集計を返す。
func (r *ReviewRunner) PromptReductionReport() ReviewPromptReductionReport {
	if r == nil {
		return ReviewPromptReductionReport{}
	}
	return r.promptReductionStats.reportValue()
}

func (r *ReviewRunner) reviewStateSummaryPrompt(input reviewStateSummaryInput) string {
	if r == nil || normalizeReviewPromptReductionMode(r.promptReductionMode) != ReviewPromptReductionModeApply {
		return ""
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	}
	summary := buildReviewStateSummary(input)
	text := summary.PromptText()
	if strings.TrimSpace(text) == "" {
		return ""
	}
	absorbedCount := len(summary.AbsorbedIntermediateRefs)
	r.promptReductionStats.recordStateSummary(absorbedCount)
	if absorbedCount == 0 {
		r.promptReductionStats.recordKeepReason("review_state_summary_current_only")
	}
	if r.promptReductionState == nil {
		r.promptReductionState = &ReviewPromptReductionState{Mode: normalizeReviewPromptReductionMode(r.promptReductionMode)}
	}
	r.promptReductionState.Summary = summary
	r.promptReductionState.Report = r.promptReductionStats.reportValue()
	return text
}

func (r *ReviewRunner) validate() error {
	if r == nil {
		return errors.New("review runner is nil")
	}
	return validateReviewRunnerOptions(ReviewRunnerOptions{
		EvidenceBuilder: r.evidenceBuilder,
		ProbeRunner:     r.probeRunner,
		Model:           r.model,
	})
}

func validateReviewRunnerOptions(opts ReviewRunnerOptions) error {
	if opts.Model == nil {
		return errReviewRunnerModelNil
	}
	if opts.EvidenceBuilder == nil {
		return errReviewRunnerEvidenceBuilderNil
	}
	if opts.ProbeRunner == nil {
		return errReviewRunnerProbeRunnerNil
	}
	return nil
}

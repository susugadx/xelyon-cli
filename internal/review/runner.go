package review

import (
	"context"
	"errors"
	"fmt"
	"io"

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
}

// ReviewRunner は /review current_changes の evidence、model、probe、report を順に束ねる。
type ReviewRunner struct {
	evidenceBuilder ReviewEvidenceProvider
	probeRunner     ReviewProbeExecutor
	model           ReviewModel

	artifactWriter        ReviewRunArtifactWriter
	artifactWarningWriter io.Writer
	progressSink          ReviewProgressSink
}

// NewReviewRunner は ReviewRunner を構築し、必須依存を検証する。
func NewReviewRunner(opts ReviewRunnerOptions) (*ReviewRunner, error) {
	if err := validateReviewRunnerOptions(opts); err != nil {
		return nil, err
	}
	return &ReviewRunner{
		evidenceBuilder:       opts.EvidenceBuilder,
		probeRunner:           opts.ProbeRunner,
		model:                 opts.Model,
		artifactWriter:        opts.ArtifactWriter,
		artifactWarningWriter: opts.ArtifactWarningWriter,
		progressSink:          opts.ProgressSink,
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
	bundle, evidenceMarkdown, evidenceRedactor = r.collectPostPass1WebSearchEvidence(ctx, bundle, plan, evidenceMarkdown, evidenceRedactor)
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

	return r.completeReviewReport(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, bundle.WebSearchEvidence.ExternalDocs)
}

func (r *ReviewRunner) collectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan ReviewProbePlan, evidenceMarkdown string, redactor reviewRunnerPromptRedactor) (ReviewEvidenceBundle, string, reviewRunnerPromptRedactor) {
	if !bundle.WebSearchEvidence.Enabled {
		return bundle, evidenceMarkdown, redactor
	}
	provider, ok := r.evidenceBuilder.(ReviewPostPass1WebSearchEvidenceProvider)
	if !ok {
		return bundle, evidenceMarkdown, redactor
	}
	bundle.WebSearchEvidence = provider.CollectPostPass1WebSearchEvidence(ctx, bundle, plan)
	evidenceMarkdown = RenderReviewEvidenceMarkdown(bundle)
	redactor = newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence_post_pass1.md", evidenceMarkdown, redactor)
	r.saveReviewRunJSONArtifact("web_search_evidence_post_pass1.json", bundle.WebSearchEvidence, redactor)
	return bundle, evidenceMarkdown, redactor
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

func (r *ReviewRunner) completeReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, externalDocs []ReviewExternalDocEvidence) (ReviewReport, error) {
	r.emitProgressRunning(reviewProgressReportItem)
	report, err := r.completeInitialReviewReport(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, externalDocs)
	if err != nil {
		r.emitProgressError(reviewProgressReportItem, err)
		return ReviewReport{}, err
	}
	r.emitProgressOK(reviewProgressReportItem, "")
	return r.completeReviewReportSaturation(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor, report, externalDocs)
}

func (r *ReviewRunner) completeInitialReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor, externalDocs []ReviewExternalDocEvidence) (ReviewReport, error) {
	reportPrompt := reviewmodelinput.BuildReportPrompt(reviewmodelinput.ReportPromptInput{
		CustomInstructions: req.CustomInstructions,
		EvidenceMarkdown:   evidenceMarkdown,
		Plan:               plan,
		ProbeSummaries:     probeSummaries,
		ProbeResults:       probeResults,
		Redactor:           redactor,
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
		ExternalDocs:          externalDocs,
	})
	if reportErr == nil {
		r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
		return report, nil
	}

	repairPrompt := reviewmodelinput.BuildReportRepairPrompt(reviewmodelinput.ReportRepairPromptInput{
		CustomInstructions:    req.CustomInstructions,
		EvidenceMarkdown:      evidenceMarkdown,
		Plan:                  plan,
		ProbeSummaries:        probeSummaries,
		ProbeResults:          probeResults,
		Redactor:              redactor,
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
		ExternalDocs:          externalDocs,
	})
	if err != nil {
		return ReviewReport{}, err
	}
	r.saveReviewRunJSONArtifact("report_final.json", report, redactor)
	return report, nil
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

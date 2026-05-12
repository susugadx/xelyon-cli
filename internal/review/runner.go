package review

import (
	"context"
	"errors"
	"fmt"
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
}

// ReviewRunner は /review current_changes の evidence、model、probe、report を順に束ねる。
type ReviewRunner struct {
	evidenceBuilder ReviewEvidenceProvider
	probeRunner     ReviewProbeExecutor
	model           ReviewModel
}

// NewReviewRunner は ReviewRunner を構築し、必須依存を検証する。
func NewReviewRunner(opts ReviewRunnerOptions) (*ReviewRunner, error) {
	if err := validateReviewRunnerOptions(opts); err != nil {
		return nil, err
	}
	return &ReviewRunner{
		evidenceBuilder: opts.EvidenceBuilder,
		probeRunner:     opts.ProbeRunner,
		model:           opts.Model,
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

	bundle, err := r.evidenceBuilder.BuildCurrentChanges(ctx)
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner build evidence: %w", err)
	}
	evidenceMarkdown := RenderReviewEvidenceMarkdown(bundle)

	plan, err := r.completeReviewProbePlan(ctx, req, evidenceMarkdown)
	if err != nil {
		return ReviewReport{}, err
	}
	probeRequests, err := BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner build probe requests: %w", err)
	}

	probeResults, err := r.runReviewProbesSequentially(ctx, probeRequests)
	if err != nil {
		return ReviewReport{}, err
	}
	probeSummaries := BuildReviewProbeSummaries(probeResults)
	redactor := newReviewRunnerPromptRedactor(bundle, probeResults)

	return r.completeReviewReport(ctx, req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor)
}

func (r *ReviewRunner) completeReviewProbePlan(ctx context.Context, req ReviewRequest, evidenceMarkdown string) (ReviewProbePlan, error) {
	planPrompt := buildReviewProbePlanPrompt(req, evidenceMarkdown)
	planResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: planPrompt,
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}

	plan, decodeErr := DecodeReviewProbePlanJSON([]byte(planResp.Content))
	if decodeErr == nil {
		return plan, nil
	}

	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase: ReviewModelPhaseProbePlan,
		Prompt: buildReviewProbePlanRepairPrompt(
			req,
			evidenceMarkdown,
			planResp.Content,
			decodeErr,
		),
	})
	if err != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner pass1 model: %w", err)
	}

	plan, decodeErr = DecodeReviewProbePlanJSON([]byte(repairResp.Content))
	if decodeErr != nil {
		return ReviewProbePlan{}, fmt.Errorf("review runner decode probe plan: %w", decodeErr)
	}
	return plan, nil
}

func (r *ReviewRunner) completeReviewReport(ctx context.Context, req ReviewRequest, evidenceMarkdown string, plan ReviewProbePlan, probeSummaries []ReviewProbeSummary, probeResults []ReviewProbeResult, redactor reviewRunnerPromptRedactor) (ReviewReport, error) {
	reportResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReport,
		Prompt: buildReviewReportPrompt(req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
	}

	report, reportErr := finalizeReviewRunnerReportModelOutput(reportResp.Content, probeSummaries, redactor)
	if reportErr == nil {
		return report, nil
	}

	repairResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase: ReviewModelPhaseReport,
		Prompt: buildReviewReportRepairPrompt(
			req,
			evidenceMarkdown,
			plan,
			probeSummaries,
			probeResults,
			redactor,
			reportResp.Content,
			reportErr,
		),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
	}

	return finalizeReviewRunnerReportModelOutput(repairResp.Content, probeSummaries, redactor)
}

func finalizeReviewRunnerReportModelOutput(content string, trustedProbeSummaries []ReviewProbeSummary, redactor reviewRunnerPromptRedactor) (ReviewReport, error) {
	report, err := decodeReviewReportStrictJSON([]byte(content))
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner decode report: %w", err)
	}
	report, err = finalizeReviewRunnerReport(report, trustedProbeSummaries, redactor)
	if err != nil {
		return ReviewReport{}, err
	}
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

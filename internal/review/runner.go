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

// ReviewRunnerOptions は ReviewRunner の依存関係を表す。
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

	planResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseProbePlan,
		Prompt: buildReviewProbePlanPrompt(req, evidenceMarkdown),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner pass1 model: %w", err)
	}

	plan, err := DecodeReviewProbePlanJSON([]byte(planResp.Content))
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner decode probe plan: %w", err)
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

	reportResp, err := r.model.CompleteReview(ctx, ReviewModelRequest{
		Phase:  ReviewModelPhaseReport,
		Prompt: buildReviewReportPrompt(req, evidenceMarkdown, plan, probeSummaries, probeResults, redactor),
	})
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner pass2 model: %w", err)
	}

	report, err := decodeReviewReportStrictJSON([]byte(reportResp.Content))
	if err != nil {
		return ReviewReport{}, fmt.Errorf("review runner decode report: %w", err)
	}
	report, err = finalizeReviewRunnerReport(report, probeSummaries, redactor)
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

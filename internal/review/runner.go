package review

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	reviewartifact "github.com/susugadx/xelyon-cli/internal/review/artifact"
	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

var (
	errReviewRunnerModelNil           = errors.New("review runner model is nil")
	errReviewRunnerEvidenceBuilderNil = errors.New("review runner evidence builder is nil")
	errReviewRunnerProbeRunnerNil     = errors.New("review runner probe runner is nil")
)

// ReviewEvidenceProvider は ReviewRunner が使う evidence 収集境界を表す。
type ReviewEvidenceProvider interface {
	BuildCurrentChanges(context.Context) (reviewevidence.ReviewEvidenceBundle, error)
}

// ReviewProbeExecutor は ReviewRunner が使う probe 実行境界を表す。
type ReviewProbeExecutor interface {
	Run(context.Context, reviewprobe.ReviewProbeRequest) (reviewprobe.ReviewProbeResult, error)
}

// ReviewRunnerOptions は ReviewRunner の必須依存関係を表す。
// concrete 依存の選択と構築は adapter/factory 側の責務にし、
// ReviewRunner は注入済みの review domain 境界だけを使う。
type ReviewRunnerOptions struct {
	EvidenceBuilder ReviewEvidenceProvider
	ProbeRunner     ReviewProbeExecutor
	Model           ReviewModel

	ArtifactWriter        reviewartifact.ReviewRunArtifactWriter
	ArtifactWarningWriter io.Writer
	ProgressSink          ReviewProgressSink
	PromptReductionMode   reviewpromptreduction.ReviewPromptReductionMode

	RawOutputArtifactsMode            reviewpromptreduction.ReviewRawOutputArtifactsMode
	RawOutputArtifactStore            reviewpromptreduction.ReviewRawOutputArtifactStore
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

	artifactWriter                    reviewartifact.ReviewRunArtifactWriter
	artifactWarningWriter             io.Writer
	progressSink                      ReviewProgressSink
	promptReductionMode               reviewpromptreduction.ReviewPromptReductionMode
	rawOutputArtifactsMode            reviewpromptreduction.ReviewRawOutputArtifactsMode
	rawOutputArtifactStore            reviewpromptreduction.ReviewRawOutputArtifactStore
	rawOutputSessionID                string
	reviewRunID                       string
	rawOutputRehydrateBudgetTokens    int
	rawOutputRehydrateBudgetMaxTokens int
	promptReductionStats              *reviewpromptreduction.Stats
	promptReductionState              *reviewpromptreduction.ReviewPromptReductionState
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
		promptReductionMode:               reviewpromptreduction.NormalizeReviewPromptReductionMode(opts.PromptReductionMode),
		rawOutputArtifactsMode:            reviewpromptreduction.NormalizeReviewRawOutputArtifactsMode(opts.RawOutputArtifactsMode),
		rawOutputArtifactStore:            opts.RawOutputArtifactStore,
		rawOutputSessionID:                strings.TrimSpace(opts.RawOutputSessionID),
		reviewRunID:                       normalizeReviewRunID(opts.ReviewRunID),
		rawOutputRehydrateBudgetTokens:    opts.RawOutputRehydrateBudgetTokens,
		rawOutputRehydrateBudgetMaxTokens: opts.RawOutputRehydrateBudgetMaxTokens,
	}, nil
}

// Run は current_changes review の MVP orchestration を実行する。
func (r *ReviewRunner) Run(ctx context.Context, req ReviewRequest) (reviewreport.ReviewReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.validate(); err != nil {
		return reviewreport.ReviewReport{}, err
	}
	r.resetPromptReductionStats()
	if req.TargetKind != reviewdomain.TargetCurrentChanges {
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner target_kind must be %q: got %q", reviewdomain.TargetCurrentChanges, req.TargetKind)
	}

	r.emitProgressRunning(reviewProgressEvidenceItem)
	bundle, err := r.evidenceBuilder.BuildCurrentChanges(ctx)
	if err != nil {
		r.emitProgressError(reviewProgressEvidenceItem, err)
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner build evidence: %w", err)
	}
	r.emitProgressOK(reviewProgressEvidenceItem, reviewEvidenceProgressDetail(bundle))
	evidenceMarkdown := reviewevidence.RenderReviewEvidenceMarkdown(bundle)
	evidenceRedactor := newReviewRunnerPromptRedactor(bundle, nil)
	r.saveReviewRunTextArtifact("evidence.md", evidenceMarkdown, evidenceRedactor)
	if bundle.WebSearchEvidence.Enabled {
		r.saveReviewRunJSONArtifact("web_search_evidence.json", bundle.WebSearchEvidence, evidenceRedactor)
	}

	r.emitProgressRunning(reviewProgressProbePlanItem)
	plan, err := r.completeReviewProbePlan(ctx, req, evidenceMarkdown, bundle)
	if err != nil {
		r.emitProgressError(reviewProgressProbePlanItem, err)
		return reviewreport.ReviewReport{}, err
	}
	bundle, evidenceMarkdown, evidenceRedactor, coverageAuditContext := r.collectPostPass1WebSearchEvidence(ctx, bundle, plan, evidenceMarkdown, evidenceRedactor)
	probeRequests, err := reviewprobe.BuildReviewProbeRequestsFromPlan(plan)
	if err != nil {
		r.emitProgressError(reviewProgressProbePlanItem, err)
		return reviewreport.ReviewReport{}, fmt.Errorf("review runner build probe requests: %w", err)
	}
	r.emitProgressOK(reviewProgressProbePlanItem, reviewProgressProbeCountDetail(len(probeRequests)))
	r.saveReviewRunJSONArtifact("probe_requests.json", probeRequests, evidenceRedactor)

	probeResults, err := r.runReviewProbesSequentially(ctx, probeRequests)
	if err != nil {
		return reviewreport.ReviewReport{}, err
	}
	probeSummaries := reviewprobe.BuildReviewProbeSummaries(probeResults)
	redactor := newReviewRunnerPromptRedactor(bundle, probeResults)
	r.saveReviewRunJSONArtifact("probe_results.json", reviewmodelinput.BuildProbeResultPromptContexts(probeResults, redactor), redactor)

	reportEvidenceMarkdown := r.reviewPromptEvidenceMarkdown(bundle, evidenceMarkdown)
	return r.completeReviewReport(ctx, req, reportEvidenceMarkdown, plan, probeSummaries, probeResults, redactor, bundle, coverageAuditContext)
}

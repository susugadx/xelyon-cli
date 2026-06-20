package reviewadapter

import (
	"context"
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	reviewartifact "github.com/susugadx/xelyon-cli/internal/review/artifact"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	"github.com/susugadx/xelyon-cli/internal/tools"
	searchtool "github.com/susugadx/xelyon-cli/internal/tools/search"
)

// RunnerFactoryOptions は ReviewRunner の concrete 依存を組み立てる入力を表す。
type RunnerFactoryOptions struct {
	RepoRoot string
	CWD      string
	Config   *config.Config

	MainProvider          string
	MainProviderConfigKey string
	MainModel             string

	// Model は provider/agent 側で実装される必須境界。
	Model review.ReviewModel

	// EvidenceBuilder / ProbeRunner / ArtifactWriter はテストや review runtime hook 用の差し替え口。
	// artifact 保存の有効化判断は agent 側に置き、この factory は注入された境界だけを渡す。
	EvidenceBuilder review.ReviewEvidenceProvider
	ProbeRunner     review.ReviewProbeExecutor

	ArtifactWriter                    reviewartifact.ReviewRunArtifactWriter
	ArtifactWarningWriter             io.Writer
	ProgressSink                      review.ReviewProgressSink
	UsageAttribution                  tools.UsageAttributionCallback
	PromptReductionMode               reviewpromptreduction.ReviewPromptReductionMode
	RawOutputArtifactsMode            reviewpromptreduction.ReviewRawOutputArtifactsMode
	RawOutputArtifactStore            reviewpromptreduction.ReviewRawOutputArtifactStore
	RawOutputSessionID                string
	RawOutputRehydrateBudgetTokens    int
	RawOutputRehydrateBudgetMaxTokens int
}

// RunnerFactory は ReviewRunner の構築責務を review domain の外側で保持する。
type RunnerFactory struct {
	opts RunnerFactoryOptions
}

// NewRunnerFactory は ReviewRunner factory を構築する。
func NewRunnerFactory(opts RunnerFactoryOptions) RunnerFactory {
	return RunnerFactory{opts: opts}
}

// NewReviewRunner は未指定の concrete 依存を補って ReviewRunner を構築する。
func (f RunnerFactory) NewReviewRunner() (*review.ReviewRunner, error) {
	evidenceBuilder := f.opts.EvidenceBuilder
	if evidenceBuilder == nil {
		evidenceBuilder = reviewevidence.NewReviewEvidenceBuilder(
			f.opts.RepoRoot,
			f.opts.CWD,
			f.reviewEvidenceBuilderOptions()...,
		)
	}

	probeRunner := f.opts.ProbeRunner
	if probeRunner == nil {
		probeRunner = reviewprobe.NewProbeRunner(f.opts.RepoRoot)
	}

	return review.NewReviewRunner(review.ReviewRunnerOptions{
		EvidenceBuilder:                   evidenceBuilder,
		ProbeRunner:                       probeRunner,
		Model:                             f.opts.Model,
		ArtifactWriter:                    f.opts.ArtifactWriter,
		ArtifactWarningWriter:             f.opts.ArtifactWarningWriter,
		ProgressSink:                      f.opts.ProgressSink,
		PromptReductionMode:               f.opts.PromptReductionMode,
		RawOutputArtifactsMode:            f.opts.RawOutputArtifactsMode,
		RawOutputArtifactStore:            f.opts.RawOutputArtifactStore,
		RawOutputSessionID:                f.opts.RawOutputSessionID,
		RawOutputRehydrateBudgetTokens:    f.opts.RawOutputRehydrateBudgetTokens,
		RawOutputRehydrateBudgetMaxTokens: f.opts.RawOutputRehydrateBudgetMaxTokens,
	})
}

func (f RunnerFactory) reviewEvidenceBuilderOptions() []reviewevidence.ReviewEvidenceBuilderOption {
	cfg := f.opts.Config
	if cfg == nil || !cfg.Review.WebSearchEvidence.Enabled {
		return nil
	}
	searcher := newReviewWebSearchRunner(f.opts)
	collector := reviewevidence.NewReviewWebSearchEvidenceCollector(reviewevidence.ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         cfg.Review.WebSearchEvidence.MaxQueries,
		MaxResultsPerQuery: cfg.Review.WebSearchEvidence.MaxResultsPerQuery,
		Searcher:           searcher,
	})
	return []reviewevidence.ReviewEvidenceBuilderOption{
		reviewevidence.WithReviewWebSearchEvidenceProvider(collector),
	}
}

type reviewWebSearchRunner struct {
	cfg                   *config.Config
	mainProvider          string
	mainProviderConfigKey string
	mainModel             string
	usageAttribution      tools.UsageAttributionCallback
}

func newReviewWebSearchRunner(opts RunnerFactoryOptions) reviewWebSearchRunner {
	return reviewWebSearchRunner{
		cfg:                   opts.Config,
		mainProvider:          opts.MainProvider,
		mainProviderConfigKey: opts.MainProviderConfigKey,
		mainModel:             opts.MainModel,
		usageAttribution:      opts.UsageAttribution,
	}
}

func (r reviewWebSearchRunner) SearchReviewWeb(ctx context.Context, query string, maxResults int) (externaldoc.WebSearchQueryResult, error) {
	resp, err := searchtool.SearchWeb(ctx, searchtool.WebSearchRequest{
		Config:                r.cfg,
		MainProvider:          r.mainProvider,
		MainProviderConfigKey: r.mainProviderConfigKey,
		MainModel:             r.mainModel,
		Query:                 query,
		MaxResults:            maxResults,
		UsageAttribution:      r.usageAttribution,
	})
	results := make([]externaldoc.WebSearchEvidenceResult, 0, len(resp.Results))
	for _, result := range resp.Results {
		results = append(results, externaldoc.WebSearchEvidenceResult{
			Title:        result.Title,
			URL:          result.URL,
			Snippet:      result.Snippet,
			SourceDomain: result.SourceDomain,
		})
	}
	return externaldoc.WebSearchQueryResult{
		Provider:  resp.Provider,
		Results:   results,
		Truncated: resp.ResultsTruncated,
	}, err
}

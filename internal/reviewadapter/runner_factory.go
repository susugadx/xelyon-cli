package reviewadapter

import "github.com/susugadx/xelyon-cli/internal/review"

// RunnerFactoryOptions は ReviewRunner の concrete 依存を組み立てる入力を表す。
type RunnerFactoryOptions struct {
	RepoRoot string
	CWD      string

	// Model は provider/agent 側で実装される必須境界。
	Model review.ReviewModel

	// EvidenceBuilder と ProbeRunner はテストや将来の hook 用の差し替え口。
	// raw probe result や trace hook はまだこの adapter の public contract にしない。
	EvidenceBuilder review.ReviewEvidenceProvider
	ProbeRunner     review.ReviewProbeExecutor
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
		evidenceBuilder = review.NewReviewEvidenceBuilder(f.opts.RepoRoot, f.opts.CWD)
	}

	probeRunner := f.opts.ProbeRunner
	if probeRunner == nil {
		probeRunner = review.NewProbeRunner(f.opts.RepoRoot)
	}

	return review.NewReviewRunner(review.ReviewRunnerOptions{
		EvidenceBuilder: evidenceBuilder,
		ProbeRunner:     probeRunner,
		Model:           f.opts.Model,
	})
}

package evidence

import (
	"context"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

// ReviewEvidenceBuilderOption は ReviewEvidenceBuilder の差し替え設定を表す。
type ReviewEvidenceBuilderOption func(*ReviewEvidenceBuilder)

// ReviewWebSearchEvidenceProvider は ReviewEvidenceBuilder が使う外部 Web 検索 evidence 境界。
type ReviewWebSearchEvidenceProvider interface {
	CollectWebSearchEvidence(context.Context, ReviewEvidenceBundle) ReviewWebSearchEvidence
}

// ReviewPostPass1WebSearchEvidenceProvider は Pass1 probe plan 後の追加 Web 検索 evidence 収集境界。
type ReviewPostPass1WebSearchEvidenceProvider interface {
	CollectPostPass1WebSearchEvidence(context.Context, ReviewEvidenceBundle, reviewprobe.ReviewProbePlan) ReviewWebSearchEvidence
}

// ReviewEvidenceBuilder は current_changes の一次情報を git と filesystem から収集する。
type ReviewEvidenceBuilder struct {
	repoRoot          string
	cwd               string
	limits            ReviewEvidenceLimits
	runner            ReviewEvidenceCommandRunner
	webSearchEvidence ReviewWebSearchEvidenceProvider
}

// WithReviewEvidenceLimits は EvidenceBuilder の resource budget を差し替える。
func WithReviewEvidenceLimits(limits ReviewEvidenceLimits) ReviewEvidenceBuilderOption {
	return func(b *ReviewEvidenceBuilder) {
		b.limits = limits
	}
}

// WithReviewEvidenceCommandRunner は EvidenceBuilder の git command runner を差し替える。
func WithReviewEvidenceCommandRunner(runner ReviewEvidenceCommandRunner) ReviewEvidenceBuilderOption {
	return func(b *ReviewEvidenceBuilder) {
		b.runner = runner
	}
}

// WithReviewWebSearchEvidenceProvider は外部 Web 検索 evidence 収集境界を差し替える。
func WithReviewWebSearchEvidenceProvider(provider ReviewWebSearchEvidenceProvider) ReviewEvidenceBuilderOption {
	return func(b *ReviewEvidenceBuilder) {
		b.webSearchEvidence = provider
	}
}

// NewReviewEvidenceBuilder は repo root と /review 起動 cwd を基準に EvidenceBuilder を構築する。
// cwd は bundle に残す診断 context であり、git 実行境界は repo root を信頼境界にする。
func NewReviewEvidenceBuilder(repoRoot, cwd string, opts ...ReviewEvidenceBuilderOption) *ReviewEvidenceBuilder {
	b := &ReviewEvidenceBuilder{
		repoRoot: repoRoot,
		cwd:      cwd,
		limits:   DefaultReviewEvidenceLimits(),
		runner:   reviewEvidenceGitRunner{},
	}
	for _, opt := range opts {
		opt(b)
	}
	b.limits = normalizeReviewEvidenceLimits(b.limits)
	if b.runner == nil {
		b.runner = reviewEvidenceGitRunner{}
	}
	return b
}

// BuildCurrentChanges は current_changes review 用の evidence bundle を構築する。
// repoRoot は review 対象 Git repo、cwd は /review 起動位置として記録する。
func (b *ReviewEvidenceBuilder) BuildCurrentChanges(ctx context.Context) (ReviewEvidenceBundle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot, cwd, err := resolveReviewEvidenceDirs(b.repoRoot, b.cwd)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	gitCollector := reviewGitEvidenceCollector{
		runner: b.runner,
		limits: b.limits,
	}
	gitEvidence, err := gitCollector.collectCurrentChanges(ctx, repoRoot, cwd)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	fileCollector := reviewFileEvidenceCollector{
		limits: b.limits,
	}
	fileEvidence, err := fileCollector.collectCurrentChanges(ctx, reviewFileEvidenceCollectionInput{
		repoRoot:              repoRoot,
		changedFiles:          gitEvidence.changedFiles,
		untrackedPaths:        gitEvidence.untrackedPaths,
		relatedCandidatePaths: gitEvidence.relatedCandidatePaths,
	})
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	bundle := buildReviewCurrentChangesBundle(repoRoot, cwd, gitEvidence, fileEvidence, b.limits)
	bundle.GenericImpactCandidates = BuildReviewGenericImpactCandidates(bundle)
	if b.webSearchEvidence != nil {
		bundle.WebSearchEvidence = b.webSearchEvidence.CollectWebSearchEvidence(ctx, bundle)
	}
	return bundle, nil
}

// CollectPostPass1WebSearchEvidence は Pass1 probe plan から追加 Web 検索 evidence を収集する。
func (b *ReviewEvidenceBuilder) CollectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan reviewprobe.ReviewProbePlan) ReviewWebSearchEvidence {
	if ctx == nil {
		ctx = context.Background()
	}
	if b == nil || b.webSearchEvidence == nil {
		return bundle.WebSearchEvidence
	}
	provider, ok := b.webSearchEvidence.(ReviewPostPass1WebSearchEvidenceProvider)
	if !ok {
		return bundle.WebSearchEvidence
	}
	return provider.CollectPostPass1WebSearchEvidence(ctx, bundle, plan)
}

func buildReviewCurrentChangesBundle(repoRoot, cwd string, git reviewCurrentChangesGitEvidence, files reviewCurrentChangesFileEvidence, limits ReviewEvidenceLimits) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind:                           TargetCurrentChanges,
		RepoRoot:                             repoRoot,
		CWD:                                  cwd,
		StatusShort:                          git.statusShort,
		StatusShortTruncated:                 git.statusShortTruncated,
		Diffs:                                git.diffs,
		ChangedFiles:                         git.changedFiles,
		ChangedFileContext:                   files.changedFileContext,
		RelatedContextFiles:                  files.relatedContextFiles,
		RelatedSearchHits:                    files.relatedSearchHits,
		GenericImpactCandidatePaths:          git.genericImpactCandidatePaths,
		GenericImpactCandidateListTruncated:  git.genericImpactListTruncated,
		GenericImpactCandidatePathsCollected: true,
		UntrackedFiles:                       files.untrackedFiles,
		RelatedCandidateListTruncated:        git.relatedCandidateListTruncated,
		RelatedSearchTruncated:               files.relatedSearchTruncated,
		UntrackedListTruncated:               git.untrackedListTruncated,
		UntrackedSnapshotsTruncated:          files.untrackedSnapshotsTruncated,
		RuleFiles:                            files.ruleFiles,
		Inventory:                            buildReviewChangeInventory(git.changedFiles, git.untrackedPaths),
		Limits:                               limits,
	}
}

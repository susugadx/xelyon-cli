package review

import "context"

// ReviewEvidenceBuilderOption は ReviewEvidenceBuilder の差し替え設定を表す。
type ReviewEvidenceBuilderOption func(*ReviewEvidenceBuilder)

// ReviewEvidenceBuilder は current_changes の一次情報を git と filesystem から収集する。
type ReviewEvidenceBuilder struct {
	repoRoot string
	cwd      string
	limits   ReviewEvidenceLimits
	runner   ReviewEvidenceCommandRunner
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
	fileEvidence, err := fileCollector.collectCurrentChanges(reviewFileEvidenceCollectionInput{
		repoRoot:       repoRoot,
		untrackedPaths: gitEvidence.untrackedPaths,
	})
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	return buildReviewCurrentChangesBundle(repoRoot, cwd, gitEvidence, fileEvidence, b.limits), nil
}

func buildReviewCurrentChangesBundle(repoRoot, cwd string, git reviewCurrentChangesGitEvidence, files reviewCurrentChangesFileEvidence, limits ReviewEvidenceLimits) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind:                  TargetCurrentChanges,
		RepoRoot:                    repoRoot,
		CWD:                         cwd,
		StatusShort:                 git.statusShort,
		StatusShortTruncated:        git.statusShortTruncated,
		Diffs:                       git.diffs,
		ChangedFiles:                git.changedFiles,
		UntrackedFiles:              files.untrackedFiles,
		UntrackedListTruncated:      git.untrackedListTruncated,
		UntrackedSnapshotsTruncated: files.untrackedSnapshotsTruncated,
		RuleFiles:                   files.ruleFiles,
		Inventory:                   buildReviewChangeInventory(git.changedFiles, git.untrackedPaths),
		Limits:                      limits,
	}
}

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

// NewReviewEvidenceBuilder は repo root と cwd を基準に EvidenceBuilder を構築する。
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
func (b *ReviewEvidenceBuilder) BuildCurrentChanges(ctx context.Context) (ReviewEvidenceBundle, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot, cwd, err := resolveReviewEvidenceDirs(b.repoRoot, b.cwd)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	statusShort, statusShortTruncated, err := b.runGit(ctx, repoRoot, cwd, reviewStatusShortGitArgs()...)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	unstagedDiff, err := b.buildDiffEvidence(ctx, repoRoot, cwd, reviewDiffEvidenceSourceUnstaged, false)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}
	stagedDiff, err := b.buildDiffEvidence(ctx, repoRoot, cwd, reviewDiffEvidenceSourceStaged, true)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	untrackedList, untrackedListTruncated, err := b.runGit(ctx, repoRoot, cwd, reviewUntrackedListGitArgs()...)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}
	untrackedPaths := parseReviewEvidenceNULPaths(untrackedList, untrackedListTruncated)
	if err := validateReviewEvidenceRelativePaths(repoRoot, untrackedPaths, "untracked path"); err != nil {
		return ReviewEvidenceBundle{}, err
	}

	changedFiles := buildReviewChangedFiles(
		unstagedDiff.nameStatusEntries,
		stagedDiff.nameStatusEntries,
	)
	untrackedEvidence, err := buildReviewUntrackedFileEvidence(repoRoot, untrackedPaths, b.limits)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}
	ruleFiles, err := buildReviewRuleFileEvidence(repoRoot, b.limits)
	if err != nil {
		return ReviewEvidenceBundle{}, err
	}

	return ReviewEvidenceBundle{
		TargetKind:                  TargetCurrentChanges,
		RepoRoot:                    repoRoot,
		CWD:                         cwd,
		StatusShort:                 statusShort,
		StatusShortTruncated:        statusShortTruncated,
		Diffs:                       []ReviewDiffEvidence{unstagedDiff.evidence, stagedDiff.evidence},
		ChangedFiles:                changedFiles,
		UntrackedFiles:              untrackedEvidence.Files,
		UntrackedListTruncated:      untrackedListTruncated,
		UntrackedSnapshotsTruncated: untrackedEvidence.SnapshotsTruncated,
		RuleFiles:                   ruleFiles,
		Inventory:                   buildReviewChangeInventory(changedFiles, untrackedPaths),
		Limits:                      b.limits,
	}, nil
}

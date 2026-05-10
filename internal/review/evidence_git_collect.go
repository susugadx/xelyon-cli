package review

import "context"

type reviewGitEvidenceCollector struct {
	runner ReviewEvidenceCommandRunner
	limits ReviewEvidenceLimits
}

type reviewCurrentChangesGitEvidence struct {
	statusShort                   string
	statusShortTruncated          bool
	diffs                         []ReviewDiffEvidence
	changedFiles                  []ReviewChangedFile
	untrackedPaths                []string
	relatedCandidatePaths         []string
	relatedCandidateListTruncated bool
	untrackedListTruncated        bool
}

type reviewDiffEvidenceResult struct {
	evidence          ReviewDiffEvidence
	nameStatusEntries []reviewNameStatusEntry
}

func (c reviewGitEvidenceCollector) collectCurrentChanges(ctx context.Context, repoRoot, cwd string) (reviewCurrentChangesGitEvidence, error) {
	statusShort, statusShortTruncated, err := c.runGit(ctx, repoRoot, cwd, reviewStatusShortGitArgs()...)
	if err != nil {
		return reviewCurrentChangesGitEvidence{}, err
	}

	unstagedDiff, err := c.buildDiffEvidence(ctx, repoRoot, cwd, reviewDiffEvidenceSourceUnstaged, false)
	if err != nil {
		return reviewCurrentChangesGitEvidence{}, err
	}
	stagedDiff, err := c.buildDiffEvidence(ctx, repoRoot, cwd, reviewDiffEvidenceSourceStaged, true)
	if err != nil {
		return reviewCurrentChangesGitEvidence{}, err
	}

	untrackedPaths, untrackedListTruncated, err := c.collectPathList(ctx, repoRoot, cwd, reviewUntrackedListGitArgs(), "untracked path")
	if err != nil {
		return reviewCurrentChangesGitEvidence{}, err
	}

	relatedCandidatePaths, relatedCandidateListTruncated, err := c.collectPathList(ctx, repoRoot, cwd, reviewRelatedCandidateListGitArgs(), "related candidate path")
	if err != nil {
		return reviewCurrentChangesGitEvidence{}, err
	}

	changedFiles := buildReviewChangedFiles(
		unstagedDiff.nameStatusEntries,
		stagedDiff.nameStatusEntries,
	)

	return reviewCurrentChangesGitEvidence{
		statusShort:                   statusShort,
		statusShortTruncated:          statusShortTruncated,
		diffs:                         []ReviewDiffEvidence{unstagedDiff.evidence, stagedDiff.evidence},
		changedFiles:                  changedFiles,
		untrackedPaths:                untrackedPaths,
		relatedCandidatePaths:         normalizeReviewRelatedCandidatePaths(relatedCandidatePaths),
		relatedCandidateListTruncated: relatedCandidateListTruncated,
		untrackedListTruncated:        untrackedListTruncated,
	}, nil
}

func (c reviewGitEvidenceCollector) collectPathList(ctx context.Context, repoRoot, cwd string, args []string, label string) ([]string, bool, error) {
	output, truncated, err := c.runGit(ctx, repoRoot, cwd, args...)
	if err != nil {
		return nil, false, err
	}
	paths := parseReviewEvidenceNULPaths(output, truncated)
	if err := validateReviewEvidenceRelativePaths(repoRoot, paths, label); err != nil {
		return nil, false, err
	}
	return paths, truncated, nil
}

func (c reviewGitEvidenceCollector) buildDiffEvidence(ctx context.Context, repoRoot, cwd, source string, staged bool) (reviewDiffEvidenceResult, error) {
	statArgs := reviewDiffMetadataGitArgs(staged, "--stat")
	nameStatusArgs := reviewDiffMetadataGitArgs(staged, "--name-status", "-z")
	diffArgs := reviewDiffBodyGitArgs(staged)

	stat, statTruncated, err := c.runGit(ctx, repoRoot, cwd, statArgs...)
	if err != nil {
		return reviewDiffEvidenceResult{}, err
	}
	nameStatus, nameStatusTruncated, err := c.runGit(ctx, repoRoot, cwd, nameStatusArgs...)
	if err != nil {
		return reviewDiffEvidenceResult{}, err
	}
	nameStatusEntries := parseReviewNameStatusEntries(nameStatus, nameStatusTruncated)
	diff, diffTruncated, err := c.runGit(ctx, repoRoot, cwd, diffArgs...)
	if err != nil {
		return reviewDiffEvidenceResult{}, err
	}

	return reviewDiffEvidenceResult{
		evidence: ReviewDiffEvidence{
			Source:              source,
			Stat:                stat,
			StatTruncated:       statTruncated,
			NameStatus:          formatReviewNameStatusEntries(nameStatusEntries),
			NameStatusTruncated: nameStatusTruncated,
			Diff:                diff,
			DiffTruncated:       diffTruncated,
		},
		nameStatusEntries: nameStatusEntries,
	}, nil
}

func (c reviewGitEvidenceCollector) runGit(ctx context.Context, repoRoot, cwd string, args ...string) (string, bool, error) {
	runner := c.runner
	if runner == nil {
		runner = reviewEvidenceGitRunner{}
	}

	output, truncated, err := runner.RunGit(ctx, repoRoot, cwd, args, c.limits.CommandTimeout, c.limits.MaxCommandOutputBytes)
	if err != nil {
		return "", false, err
	}
	return output, truncated, nil
}

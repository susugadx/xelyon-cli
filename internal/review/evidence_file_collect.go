package review

type reviewFileEvidenceCollector struct {
	limits ReviewEvidenceLimits
}

type reviewFileEvidenceCollectionInput struct {
	repoRoot       string
	untrackedPaths []string
}

type reviewCurrentChangesFileEvidence struct {
	untrackedFiles              []ReviewUntrackedFile
	untrackedSnapshotsTruncated bool
	ruleFiles                   []ReviewRuleFileEvidence
}

func (c reviewFileEvidenceCollector) collectCurrentChanges(input reviewFileEvidenceCollectionInput) (reviewCurrentChangesFileEvidence, error) {
	untrackedEvidence, err := buildReviewUntrackedFileEvidence(input.repoRoot, input.untrackedPaths, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}
	ruleFiles, err := buildReviewRuleFileEvidence(input.repoRoot, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}

	return reviewCurrentChangesFileEvidence{
		untrackedFiles:              untrackedEvidence.Files,
		untrackedSnapshotsTruncated: untrackedEvidence.SnapshotsTruncated,
		ruleFiles:                   ruleFiles,
	}, nil
}

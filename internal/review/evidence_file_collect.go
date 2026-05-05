package review

type reviewFileEvidenceCollector struct {
	limits ReviewEvidenceLimits
}

type reviewCurrentChangesFileEvidence struct {
	untrackedFiles              []ReviewUntrackedFile
	untrackedSnapshotsTruncated bool
	ruleFiles                   []ReviewRuleFileEvidence
}

func (c reviewFileEvidenceCollector) collectCurrentChanges(repoRoot string, untrackedPaths []string) (reviewCurrentChangesFileEvidence, error) {
	untrackedEvidence, err := buildReviewUntrackedFileEvidence(repoRoot, untrackedPaths, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}
	ruleFiles, err := buildReviewRuleFileEvidence(repoRoot, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}

	return reviewCurrentChangesFileEvidence{
		untrackedFiles:              untrackedEvidence.Files,
		untrackedSnapshotsTruncated: untrackedEvidence.SnapshotsTruncated,
		ruleFiles:                   ruleFiles,
	}, nil
}

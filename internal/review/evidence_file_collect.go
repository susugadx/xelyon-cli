package review

import (
	"context"
	pathpkg "path"
)

type reviewFileEvidenceCollector struct {
	limits ReviewEvidenceLimits
}

type reviewFileEvidenceCollectionInput struct {
	repoRoot              string
	changedFiles          []ReviewChangedFile
	untrackedPaths        []string
	relatedCandidatePaths []string
}

type reviewCurrentChangesFileEvidence struct {
	changedFileContext          []ReviewContextFileEvidence
	relatedContextFiles         []ReviewContextFileEvidence
	relatedSearchHits           []ReviewRelatedSearchHit
	relatedSearchTruncated      bool
	untrackedFiles              []ReviewUntrackedFile
	untrackedSnapshotsTruncated bool
	ruleFiles                   []ReviewRuleFileEvidence
}

func (c reviewFileEvidenceCollector) collectCurrentChanges(ctx context.Context, input reviewFileEvidenceCollectionInput) (reviewCurrentChangesFileEvidence, error) {
	contextSeeds := buildReviewContextSeedFiles(input.repoRoot, input.changedFiles, input.untrackedPaths)
	contextEvidence, err := buildReviewContextEvidence(ctx, input.repoRoot, contextSeeds, input.relatedCandidatePaths, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}
	untrackedEvidence, err := buildReviewUntrackedFileEvidence(input.repoRoot, input.untrackedPaths, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}
	ruleFiles, err := buildReviewRuleFileEvidence(input.repoRoot, c.limits)
	if err != nil {
		return reviewCurrentChangesFileEvidence{}, err
	}

	return reviewCurrentChangesFileEvidence{
		changedFileContext:          contextEvidence.changedFileContext,
		relatedContextFiles:         contextEvidence.relatedContextFiles,
		relatedSearchHits:           contextEvidence.relatedSearchHits,
		relatedSearchTruncated:      contextEvidence.relatedSearchTruncated,
		untrackedFiles:              untrackedEvidence.Files,
		untrackedSnapshotsTruncated: untrackedEvidence.SnapshotsTruncated,
		ruleFiles:                   ruleFiles,
	}, nil
}

func buildReviewContextSeedFiles(repoRoot string, changedFiles []ReviewChangedFile, untrackedPaths []string) []ReviewChangedFile {
	seeds := make([]ReviewChangedFile, 0, len(changedFiles)+len(untrackedPaths))
	seen := make(map[string]struct{}, len(changedFiles)+len(untrackedPaths))

	for _, file := range changedFiles {
		seeds = append(seeds, file)
		_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, file.Path)
		if err == nil && !reviewStatusHasPrefix(file.Status, "D") {
			seen[relPath] = struct{}{}
		}
	}

	for _, path := range untrackedPaths {
		_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
		if err != nil {
			continue
		}
		if pathpkg.Ext(relPath) != ".go" {
			continue
		}
		if _, ok := seen[relPath]; ok {
			continue
		}
		seen[relPath] = struct{}{}
		seeds = append(seeds, ReviewChangedFile{
			Path:     relPath,
			Status:   "??",
			Unstaged: true,
		})
	}

	return seeds
}

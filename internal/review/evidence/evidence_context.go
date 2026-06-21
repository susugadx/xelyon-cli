package evidence

import (
	"context"
)

const (
	reviewContextFileRoleChanged     = "changed_file"
	reviewContextFileRoleRelatedGo   = "related_go"
	reviewContextFileRoleRelatedTest = "related_test"

	reviewContextSkipGenerated           = "generated"
	reviewContextSkipVendor              = "vendor"
	reviewContextSkipSymlink             = "symlink"
	reviewContextSkipNonRegular          = "non_regular"
	reviewContextSkipBinary              = "binary"
	reviewContextSkipTooLarge            = "too_large"
	reviewContextSkipTotalBudgetExceeded = "total_budget_exceeded"
	reviewContextSkipMaxFilesExceeded    = "max_files_exceeded"
	reviewContextSkipInvalidPath         = "invalid_path"
	reviewContextSkipStatFailed          = "stat_failed"
	reviewContextSkipReadFailed          = "read_failed"
)

type reviewContextEvidence struct {
	changedFileContext     []ReviewContextFileEvidence
	relatedContextFiles    []ReviewContextFileEvidence
	relatedSearchHits      []ReviewRelatedSearchHit
	relatedSearchTruncated bool
}

type reviewContextEvidenceCollector struct {
	repoRoot string
	ctx      context.Context
	limits   ReviewEvidenceLimits

	contextFileCount              int
	totalContextRead              int64
	maxContextFilesExceededLogged bool

	changedPaths          map[string]struct{}
	relatedCandidatePaths []string
}

type reviewContextRelatedCandidate struct {
	path     string
	role     string
	priority int
}

type reviewContextChangedLanguageStem struct {
	source bool
	test   bool
}

type reviewContextChangedLanguageScope struct {
	stemsByDir map[string]map[string]reviewContextChangedLanguageStem
}

func buildReviewContextEvidence(ctx context.Context, repoRoot string, changedFiles []ReviewChangedFile, relatedCandidatePaths []string, limits ReviewEvidenceLimits) (reviewContextEvidence, error) {
	collector := newReviewContextEvidenceCollector(ctx, repoRoot, limits, changedFiles, relatedCandidatePaths)
	changedFileContext := collector.collectChangedFileContext(changedFiles)
	if err := collector.contextErr(); err != nil {
		return reviewContextEvidence{}, err
	}
	relatedContextFiles := collector.collectRelatedContextFiles(changedFiles)
	if err := collector.contextErr(); err != nil {
		return reviewContextEvidence{}, err
	}
	relatedSearchCollector := newReviewRelatedSearchCollector(collector.ctx, collector.repoRoot, collector.limits, collector.changedPaths, collector.relatedCandidatePaths)
	relatedSearch, err := relatedSearchCollector.collect(changedFileContext)
	if err != nil {
		return reviewContextEvidence{}, err
	}

	return reviewContextEvidence{
		changedFileContext:     changedFileContext,
		relatedContextFiles:    relatedContextFiles,
		relatedSearchHits:      relatedSearch.hits,
		relatedSearchTruncated: relatedSearch.truncated,
	}, nil
}

func newReviewContextEvidenceCollector(ctx context.Context, repoRoot string, limits ReviewEvidenceLimits, changedFiles []ReviewChangedFile, relatedCandidatePaths []string) *reviewContextEvidenceCollector {
	if ctx == nil {
		ctx = context.Background()
	}
	collector := &reviewContextEvidenceCollector{
		repoRoot:              repoRoot,
		ctx:                   ctx,
		limits:                normalizeReviewEvidenceLimits(limits),
		changedPaths:          make(map[string]struct{}, len(changedFiles)),
		relatedCandidatePaths: normalizeReviewRelatedCandidatePaths(relatedCandidatePaths),
	}
	for _, file := range changedFiles {
		_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, file.Path)
		if err != nil {
			continue
		}
		collector.changedPaths[relPath] = struct{}{}
	}
	return collector
}

func (c *reviewContextEvidenceCollector) contextErr() error {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

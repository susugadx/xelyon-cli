package evidence

import "context"

type reviewRelatedSearchCollectionResult struct {
	hits      []ReviewRelatedSearchHit
	truncated bool
}

type reviewRelatedSearchCollector struct {
	repoRoot string
	ctx      context.Context
	limits   ReviewEvidenceLimits

	searchFileCount int
	totalSearchRead int64
	truncated       bool

	changedPaths          map[string]struct{}
	relatedCandidatePaths []string
}

func newReviewRelatedSearchCollector(ctx context.Context, repoRoot string, limits ReviewEvidenceLimits, changedPaths map[string]struct{}, relatedCandidatePaths []string) *reviewRelatedSearchCollector {
	if ctx == nil {
		ctx = context.Background()
	}
	return &reviewRelatedSearchCollector{
		repoRoot:              repoRoot,
		ctx:                   ctx,
		limits:                normalizeReviewEvidenceLimits(limits),
		changedPaths:          changedPaths,
		relatedCandidatePaths: relatedCandidatePaths,
	}
}

func (c *reviewRelatedSearchCollector) collect(changedFileContext []ReviewContextFileEvidence) (reviewRelatedSearchCollectionResult, error) {
	termSet := buildReviewRelatedSearchTerms(changedFileContext, c.limits)
	if termSet.truncated {
		c.markTruncated()
	}
	if len(termSet.items) == 0 {
		return reviewRelatedSearchCollectionResult{
			hits:      []ReviewRelatedSearchHit{},
			truncated: c.truncated,
		}, nil
	}

	buckets := newReviewRelatedSearchHitBuckets(c.limits.MaxRelatedSearchHits)
	for i, relPath := range c.relatedCandidatePaths {
		if buckets.highestPriorityFull() {
			if i < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
		if c.searchFileCount >= c.limits.MaxRelatedSearchFiles || c.totalSearchRead >= c.limits.MaxTotalRelatedSearchBytes {
			if i < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
		if err := c.contextErr(); err != nil {
			return reviewRelatedSearchCollectionResult{}, err
		}
		data, ok := c.readCandidate(relPath)
		if !ok {
			continue
		}
		scan := c.scanRelatedSearchContent(&buckets, relPath, string(data), termSet.items)
		if scan.stop {
			if i+1 < len(c.relatedCandidatePaths) {
				c.markTruncated()
			}
			break
		}
	}
	if buckets.outputTruncated() {
		c.markTruncated()
	}
	return reviewRelatedSearchCollectionResult{
		hits:      buckets.flatten(),
		truncated: c.truncated,
	}, nil
}

func (c *reviewRelatedSearchCollector) contextErr() error {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

func (c *reviewRelatedSearchCollector) markTruncated() {
	c.truncated = true
}

func (c *reviewRelatedSearchCollector) readCandidate(relPath string) ([]byte, bool) {
	if !isReviewContextRelatedGoPath(relPath) {
		return nil, false
	}
	if _, changed := c.changedPaths[relPath]; changed {
		return nil, false
	}

	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(c.repoRoot, relPath)
	if err != nil {
		return nil, false
	}

	remainingTotal := c.limits.MaxTotalRelatedSearchBytes - c.totalSearchRead
	if remainingTotal <= 0 {
		return nil, false
	}
	maxBytes := minReviewEvidenceInt64(c.limits.MaxRelatedSearchFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot: c.repoRoot,
		absPath:  absPath,
		relPath:  relPath,
		maxBytes: maxBytes,
	})
	if file.regular {
		c.searchFileCount++
	}
	if file.status != reviewEvidenceRegularFileReadOK {
		return nil, false
	}
	c.totalSearchRead += file.readBytes
	if file.truncated {
		c.markTruncated()
	}
	if file.binary {
		return nil, false
	}
	return file.data, true
}

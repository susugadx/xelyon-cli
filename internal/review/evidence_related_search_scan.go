package review

import "strings"

type reviewRelatedSearchScanResult struct {
	stop bool
}

type reviewRelatedSearchHitBuckets struct {
	limit int
	hits  [reviewRelatedSearchPriorityCount][]ReviewRelatedSearchHit
}

func (c *reviewRelatedSearchCollector) scanRelatedSearchContent(hits *reviewRelatedSearchHitBuckets, relPath, content string, terms []reviewRelatedSearchTerm) reviewRelatedSearchScanResult {
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		if hits.highestPriorityFull() {
			if hasReviewRelatedSearchRemainingContent(lines[i:]) {
				c.markTruncated()
			}
			return reviewRelatedSearchScanResult{stop: true}
		}
		lineText := strings.TrimRight(line, "\r\n")
		if lineText == "" {
			continue
		}
		for _, term := range terms {
			if !reviewRelatedSearchTermMatchesLine(term, lineText) {
				continue
			}
			snippet := strings.TrimSpace(lineText)
			if snippet == "" {
				snippet = lineText
			}
			snippet, snippetTruncated := truncateReviewEvidenceStringPrefix(snippet, c.limits.MaxSearchSnippetBytes)
			if snippetTruncated {
				c.markTruncated()
			}
			if hits.appendHit(term.priority, ReviewRelatedSearchHit{
				Path:    relPath,
				Line:    i + 1,
				Snippet: snippet,
				Reason:  term.reason,
			}) {
				c.markTruncated()
			}
			break
		}
	}
	return reviewRelatedSearchScanResult{}
}

func newReviewRelatedSearchHitBuckets(limit int) reviewRelatedSearchHitBuckets {
	return reviewRelatedSearchHitBuckets{
		limit: limit,
	}
}

func (b *reviewRelatedSearchHitBuckets) appendHit(priority int, hit ReviewRelatedSearchHit) bool {
	if priority < 0 || priority >= reviewRelatedSearchPriorityCount {
		return false
	}
	if b.limit <= 0 {
		return true
	}
	if len(b.hits[priority]) >= b.limit {
		return true
	}
	b.hits[priority] = append(b.hits[priority], hit)
	return false
}

func (b reviewRelatedSearchHitBuckets) highestPriorityFull() bool {
	return b.limit > 0 && len(b.hits[reviewRelatedSearchPrioritySymbol]) >= b.limit
}

func (b reviewRelatedSearchHitBuckets) flatten() []ReviewRelatedSearchHit {
	result := make([]ReviewRelatedSearchHit, 0, b.limit)
	for priority := 0; priority < reviewRelatedSearchPriorityCount; priority++ {
		for _, hit := range b.hits[priority] {
			if len(result) >= b.limit {
				return result
			}
			result = append(result, hit)
		}
	}
	return result
}

func (b reviewRelatedSearchHitBuckets) outputTruncated() bool {
	total := 0
	for priority := 0; priority < reviewRelatedSearchPriorityCount; priority++ {
		total += len(b.hits[priority])
	}
	if b.limit <= 0 {
		return total > 0
	}
	return total > b.limit
}

func hasReviewRelatedSearchRemainingContent(lines []string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}

func reviewRelatedSearchTermMatchesLine(term reviewRelatedSearchTerm, lineText string) bool {
	if isReviewRelatedPackageDeclarationLine(lineText) {
		return false
	}
	if !strings.Contains(lineText, term.term) {
		return false
	}
	return true
}

func isReviewRelatedPackageDeclarationLine(lineText string) bool {
	return strings.HasPrefix(strings.TrimSpace(lineText), "package ")
}

package evidence

import (
	"strings"
)

type reviewGenericImpactReferenceSearch struct {
	role   string
	reason string
	filter func(string) bool
}

func (b *reviewGenericImpactCandidateBuilder) collectSearchCandidates() {
	for _, token := range b.tokens {
		if token.fromDiff {
			for _, search := range reviewGenericImpactDiffTokenReferenceSearches {
				b.collectTokenReferences(search, token.value)
			}
		}
		if token.fromStem {
			b.collectTokenReferences(reviewGenericImpactStemTokenReferenceSearch, token.value)
		}
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectTokenReferences(search reviewGenericImpactReferenceSearch, token string) {
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	for _, path := range b.repoPaths {
		if b.searchHitLimitReached(search.role, token) {
			b.truncated = true
			return
		}
		if b.isChangedPath(path) || !search.filter(path) {
			continue
		}
		content, ok := b.readSearchFile(path)
		if !ok {
			continue
		}
		b.collectTokenReferencesInContent(search, path, token, content)
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectTokenReferencesInContent(search reviewGenericImpactReferenceSearch, path, token, content string) {
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		if b.searchHitLimitReached(search.role, token) {
			b.truncated = true
			return
		}
		lineText := strings.TrimRight(line, "\r\n")
		if lineText == "" || !reviewGenericImpactLineContainsToken(lineText, token) {
			continue
		}
		snippet := strings.TrimSpace(lineText)
		if snippet == "" {
			snippet = lineText
		}
		var snippetTruncated bool
		snippet, snippetTruncated = truncateReviewEvidenceStringPrefix(snippet, b.limits.MaxSearchSnippetBytes)
		if snippetTruncated {
			b.truncated = true
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:    path,
			Role:    search.role,
			Reason:  search.reason,
			Token:   token,
			Line:    i + 1,
			Snippet: snippet,
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) readSearchFile(path string) (string, bool) {
	if b.searchFileCount >= b.limits.MaxRelatedSearchFiles || b.totalSearchRead >= b.limits.MaxTotalRelatedSearchBytes {
		b.truncated = true
		return "", false
	}
	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(b.repoRoot, path)
	if err != nil {
		return "", false
	}
	remainingTotal := b.limits.MaxTotalRelatedSearchBytes - b.totalSearchRead
	if remainingTotal <= 0 {
		b.truncated = true
		return "", false
	}
	maxBytes := minReviewEvidenceInt64(b.limits.MaxRelatedSearchFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot: b.repoRoot,
		absPath:  absPath,
		relPath:  relPath,
		maxBytes: maxBytes,
	})
	if file.regular {
		b.searchFileCount++
	}
	if file.status != reviewEvidenceRegularFileReadOK {
		return "", false
	}
	b.totalSearchRead += file.readBytes
	if file.truncated {
		b.truncated = true
	}
	if file.binary {
		return "", false
	}
	return string(file.data), true
}

func (b *reviewGenericImpactCandidateBuilder) searchHitLimitReached(role, token string) bool {
	return b.tokenHits[role+"\x00"+token] >= reviewGenericImpactMaxHitsPerToken
}

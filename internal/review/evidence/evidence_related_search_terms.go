package evidence

import "strings"

type reviewRelatedSearchTerm struct {
	term     string
	reason   string
	priority int
}

type reviewRelatedSearchTermSet struct {
	items     []reviewRelatedSearchTerm
	truncated bool
}

const (
	reviewRelatedSearchPrioritySymbol = iota
	reviewRelatedSearchPriorityFileStem
	reviewRelatedSearchPriorityPackage
	reviewRelatedSearchPriorityCount
)

func buildReviewRelatedSearchTerms(changedFileContext []ReviewContextFileEvidence, limits ReviewEvidenceLimits) reviewRelatedSearchTermSet {
	limits = normalizeReviewEvidenceLimits(limits)
	terms := reviewRelatedSearchTermSet{
		items: make([]reviewRelatedSearchTerm, 0, limits.MaxRelatedSearchTerms),
	}
	seen := make(map[string]struct{})

	addTerm := func(term, reason string, priority int) bool {
		term = strings.TrimSpace(term)
		if term == "" {
			return false
		}
		if _, ok := seen[term]; ok {
			return false
		}
		seen[term] = struct{}{}
		if len(terms.items) >= limits.MaxRelatedSearchTerms {
			terms.truncated = true
			return true
		}
		terms.items = append(terms.items, reviewRelatedSearchTerm{term: term, reason: reason, priority: priority})
		return false
	}

	for _, file := range changedFileContext {
		if file.Skipped {
			continue
		}
		language, ok := reviewEvidenceLanguageSpecForRelatedPath(file.Path)
		if !ok || language.extractRelatedSearchTerms == nil {
			continue
		}
		if language.extractRelatedSearchTerms(language, file, addTerm) {
			return terms
		}
	}

	return terms
}

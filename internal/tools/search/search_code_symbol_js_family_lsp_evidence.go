package search

import "github.com/susugadx/xelyon-cli/internal/navigation"

const jsFamilyLSPReferenceEvidenceLimit = maxGenericRefs

type jsFamilyLSPReferenceCandidateRef struct {
	candidate jsFamilyLSPReferenceCandidate
	summary   genericSymbolRef
}

type jsFamilyLSPReferenceCollector struct {
	def        genericSymbolDef
	opts       jsFamilyLSPReferenceOptions
	candidates []jsFamilyLSPReferenceCandidateRef
	builder    *jsFamilyLSPReferenceBuilder
}

func newJSFamilyLSPReferenceCollector(symbol string, def genericSymbolDef, opts jsFamilyLSPReferenceOptions, capacity int) *jsFamilyLSPReferenceCollector {
	if capacity < 0 {
		capacity = 0
	}
	return &jsFamilyLSPReferenceCollector{
		def:        def,
		opts:       opts,
		candidates: make([]jsFamilyLSPReferenceCandidateRef, 0, capacity),
		builder:    newJSFamilyLSPReferenceBuilder(symbol),
	}
}

func (c *jsFamilyLSPReferenceCollector) AddLocation(loc navigation.LSPLocation) bool {
	if c == nil {
		return false
	}
	candidate, ok := jsFamilyLSPReferenceCandidateFromLocation(loc, c.opts)
	if !ok {
		return false
	}
	c.candidates = append(c.candidates, jsFamilyLSPReferenceCandidateRef{
		candidate: candidate,
		summary:   c.builder.SummaryRef(candidate),
	})
	return true
}

func (c *jsFamilyLSPReferenceCollector) Result() jsFamilyLSPReferenceCollection {
	if c == nil {
		return jsFamilyLSPReferenceCollection{}
	}
	summaryRefs := make([]genericSymbolRef, 0, len(c.candidates))
	candidatesByKey := make(map[jsFamilyLSPReferenceKey]jsFamilyLSPReferenceCandidateRef, len(c.candidates))
	for _, candidate := range c.candidates {
		summaryRefs = append(summaryRefs, candidate.summary)
		candidatesByKey[jsFamilyLSPReferenceKeyForRef(candidate.summary)] = candidate
	}

	selected := selectJSFamilyLSPReferenceEvidence(c.def, summaryRefs, jsFamilyLSPReferenceEvidenceLimit)
	refs := make([]genericSymbolRef, 0, len(selected))
	for _, ref := range selected {
		candidateRef, ok := candidatesByKey[jsFamilyLSPReferenceKeyForRef(ref)]
		if !ok {
			continue
		}
		refs = append(refs, c.builder.RefWithSummary(candidateRef.candidate, ref))
	}

	return jsFamilyLSPReferenceCollection{refs: refs, summaryRefs: summaryRefs}
}

func (c *jsFamilyLSPReferenceCollector) Close() {
	if c != nil && c.builder != nil {
		c.builder.Close()
	}
}

type jsFamilyLSPReferenceKey struct {
	file string
	line int
}

func jsFamilyLSPReferenceKeyForRef(ref genericSymbolRef) jsFamilyLSPReferenceKey {
	return jsFamilyLSPReferenceKey{file: ref.File, line: ref.Line}
}

func selectJSFamilyLSPReferenceEvidence(def genericSymbolDef, refs []genericSymbolRef, limit int) []genericSymbolRef {
	if limit <= 0 || len(refs) == 0 {
		return nil
	}
	if len(refs) <= limit {
		return dedupeGenericRefs(refs)
	}

	classified := classifyJSFamilySymbolRefsFromAST(refs)
	selected := make([]genericSymbolRef, 0, limit)
	seen := make(map[jsFamilyLSPReferenceKey]struct{}, limit)
	add := func(ref genericSymbolRef) bool {
		key := jsFamilyLSPReferenceKeyForRef(ref)
		if key.file == "" || key.line <= 0 {
			return len(selected) < limit
		}
		if _, ok := seen[key]; ok {
			return len(selected) < limit
		}
		seen[key] = struct{}{}
		selected = append(selected, ref)
		return len(selected) < limit
	}
	addGroup := func(group []genericSymbolRef, groupLimit int, testOnly bool) {
		for _, ref := range prioritizeGenericRefs(def, group, groupLimit, testOnly) {
			if !add(ref) {
				return
			}
		}
	}

	addGroup(classified.imports, jsImportLimit, false)
	addGroup(classified.callers, jsCallerLimit, false)
	addGroup(classified.typeRefs, jsTypeRefLimit, false)
	addGroup(classified.others, genericRefLimit, false)
	addGroup(classified.tests, genericTestLimit, true)

	if len(selected) < limit {
		addGroup(refs, limit, false)
	}
	return selected
}

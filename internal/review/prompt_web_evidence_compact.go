package review

const reviewWebSearchDiscoveryCompactMinSavedTokens = 128
const reviewExternalDocAbsorbedCompactMinSavedTokens = 128
const reviewWebSearchEvidenceRawArtifactRef = "web_search_evidence*.json"

func (r *ReviewRunner) reviewPromptEvidenceMarkdown(bundle ReviewEvidenceBundle, rawMarkdown string) string {
	if r == nil {
		return rawMarkdown
	}
	mode := normalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == ReviewPromptReductionModeOff {
		return rawMarkdown
	}
	compactedBundle, savedBytes, savedTokens, ok := compactReviewWebSearchDiscoveryEvidence(bundle)
	if !ok {
		return rawMarkdown
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	}
	applied := mode == ReviewPromptReductionModeApply
	r.promptReductionStats.record("review_web_search_discovery", savedBytes, savedTokens, applied)
	if !applied {
		return rawMarkdown
	}
	return RenderReviewEvidenceMarkdown(compactedBundle)
}

func (r *ReviewRunner) reviewPromptEvidenceMarkdownForAbsorbedReport(phase ReviewModelPhase, bundle ReviewEvidenceBundle, rawMarkdown string, report ReviewReport) string {
	if r == nil {
		return rawMarkdown
	}
	mode := normalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == ReviewPromptReductionModeOff {
		return rawMarkdown
	}

	baseBundle := bundle
	discoverySavedBytes := 0
	discoverySavedTokens := 0
	discoveryOK := false
	if compacted, savedBytes, savedTokens, ok := compactReviewWebSearchDiscoveryEvidence(bundle); ok {
		baseBundle = compacted
		discoverySavedBytes = savedBytes
		discoverySavedTokens = savedTokens
		discoveryOK = true
	}
	compactedBundle, items, savedBytes, savedTokens, ok := compactReviewExternalDocAbsorbedEvidence(phase, baseBundle, report)
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	}
	applied := mode == ReviewPromptReductionModeApply
	if discoveryOK {
		discoveryMarkdown := RenderReviewEvidenceMarkdown(baseBundle)
		if applied && discoveryMarkdown != rawMarkdown {
			r.promptReductionStats.record("review_web_search_discovery", discoverySavedBytes, discoverySavedTokens, applied)
			if !ok {
				return discoveryMarkdown
			}
		} else if applied && !ok {
			return rawMarkdown
		}
	}
	if !ok {
		return rawMarkdown
	}
	r.promptReductionStats.record("review_external_doc_absorbed", savedBytes, savedTokens, applied)
	for _, item := range items {
		r.recordPromptReductionItem(item)
	}
	if !applied {
		return rawMarkdown
	}
	return RenderReviewEvidenceMarkdown(compactedBundle)
}

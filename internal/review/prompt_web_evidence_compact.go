package review

import (
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func (r *ReviewRunner) reviewPromptEvidenceMarkdown(bundle reviewevidence.ReviewEvidenceBundle, rawMarkdown string) string {
	if r == nil {
		return rawMarkdown
	}
	mode := reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == reviewpromptreduction.ReviewPromptReductionModeOff {
		return rawMarkdown
	}
	compactedBundle, savedBytes, savedTokens, ok := reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence(bundle)
	if !ok {
		return rawMarkdown
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(r.promptReductionMode)
	}
	applied := mode == reviewpromptreduction.ReviewPromptReductionModeApply
	r.promptReductionStats.RecordCandidate("review_web_search_discovery", savedBytes, savedTokens, applied)
	if !applied {
		return rawMarkdown
	}
	return reviewmodelinput.RenderReviewEvidenceMarkdown(compactedBundle)
}

func (r *ReviewRunner) reviewPromptEvidenceMarkdownForAbsorbedReport(phase ReviewModelPhase, bundle reviewevidence.ReviewEvidenceBundle, rawMarkdown string, report reviewreport.ReviewReport) string {
	if r == nil {
		return rawMarkdown
	}
	mode := reviewpromptreduction.NormalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == reviewpromptreduction.ReviewPromptReductionModeOff {
		return rawMarkdown
	}

	baseBundle := bundle
	discoverySavedBytes := 0
	discoverySavedTokens := 0
	discoveryOK := false
	if compacted, savedBytes, savedTokens, ok := reviewpromptreduction.CompactReviewWebSearchDiscoveryEvidence(bundle); ok {
		baseBundle = compacted
		discoverySavedBytes = savedBytes
		discoverySavedTokens = savedTokens
		discoveryOK = true
	}
	compactedBundle, items, savedBytes, savedTokens, ok := reviewpromptreduction.CompactReviewExternalDocAbsorbedEvidence(reviewPromptReductionPhase(phase), baseBundle, report)
	if r.promptReductionStats == nil {
		r.promptReductionStats = reviewpromptreduction.NewStats(r.promptReductionMode)
	}
	applied := mode == reviewpromptreduction.ReviewPromptReductionModeApply
	if discoveryOK {
		discoveryMarkdown := reviewmodelinput.RenderReviewEvidenceMarkdown(baseBundle)
		if applied && discoveryMarkdown != rawMarkdown {
			r.promptReductionStats.RecordCandidate("review_web_search_discovery", discoverySavedBytes, discoverySavedTokens, applied)
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
	r.promptReductionStats.RecordCandidate("review_external_doc_absorbed", savedBytes, savedTokens, applied)
	for _, item := range items {
		r.recordPromptReductionItem(item)
	}
	if !applied {
		return rawMarkdown
	}
	return reviewmodelinput.RenderReviewEvidenceMarkdown(compactedBundle)
}

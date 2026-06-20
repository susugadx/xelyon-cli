package promptreduction

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	"github.com/susugadx/xelyon-cli/internal/token"
)

// CompactReviewExternalDocAbsorbedEvidence は report scope に吸収済みの external_doc snippet を placeholder 化する。
func CompactReviewExternalDocAbsorbedEvidence(phase ReviewModelPhase, bundle ReviewEvidenceBundle, report ReviewReport) (ReviewEvidenceBundle, []ReviewPromptReductionItem, int, int, bool) {
	evidence := bundle.WebSearchEvidence
	if !reviewWebSearchEvidenceSafeForExternalDocAbsorption(evidence) ||
		strings.TrimSpace(report.SchemaVersion) == "" ||
		report.ScopeCoverage == nil {
		return ReviewEvidenceBundle{}, nil, 0, 0, false
	}
	safeRefs, unsafeRefs := reviewExternalDocAbsorptionRefs(report)
	if len(safeRefs) == 0 {
		return ReviewEvidenceBundle{}, nil, 0, 0, false
	}

	compacted := cloneReviewEvidenceBundleForPromptCompact(bundle)
	items := make([]ReviewPromptReductionItem, 0)
	originalBytes := 0
	replacementBytes := 0
	changed := false
	for docIndex := range compacted.WebSearchEvidence.ExternalDocs {
		doc := &compacted.WebSearchEvidence.ExternalDocs[docIndex]
		if !reviewExternalDocSafeForAbsorbedPrompt(*doc) {
			continue
		}
		for snippetIndex := range doc.Snippets {
			snippet := &doc.Snippets[snippetIndex]
			key := reviewExternalDocSnippetAbsorptionKey(doc.DocID, snippet.SnippetID)
			refSummary, ok := safeRefs[key]
			if !ok || len(refSummary.owners) == 0 {
				continue
			}
			if _, unsafe := unsafeRefs[key]; unsafe {
				continue
			}
			if !refSummary.matches(*doc, *snippet) {
				continue
			}
			if !reviewExternalDocSnippetSafeForAbsorbedPrompt(*snippet) {
				continue
			}
			replacement := reviewExternalDocAbsorbedSnippetPlaceholder(*doc, *snippet, refSummary.owners)
			if len(replacement) >= len(snippet.Content) {
				continue
			}
			snippetOriginalBytes := len(snippet.Content)
			snippetReplacementBytes := len(replacement)
			originalBytes += snippetOriginalBytes
			replacementBytes += snippetReplacementBytes
			items = append(items, reviewExternalDocAbsorbedPromptReductionItem(
				phase,
				*doc,
				*snippet,
				refSummary.owners,
				snippetOriginalBytes,
				snippetReplacementBytes,
			))
			snippet.Content = replacement
			changed = true
		}
	}
	if !changed {
		return ReviewEvidenceBundle{}, nil, 0, 0, false
	}
	savedBytes := originalBytes - replacementBytes
	savedTokens := token.EstimateTokenCount(strings.Repeat("x", originalBytes)) - token.EstimateTokenCount(strings.Repeat("x", replacementBytes))
	if savedBytes <= 0 || savedTokens < reviewExternalDocAbsorbedCompactMinSavedTokens {
		return ReviewEvidenceBundle{}, nil, 0, 0, false
	}
	return compacted, items, savedBytes, savedTokens, true
}

func reviewWebSearchEvidenceSafeForExternalDocAbsorption(evidence ReviewWebSearchEvidence) bool {
	if !evidence.Enabled ||
		strings.TrimSpace(evidence.Error) != "" ||
		evidence.Truncated ||
		evidence.Inconclusive ||
		len(evidence.ExternalDocs) == 0 {
		return false
	}
	support := externaldoc.SummarizeExternalSupport(evidence)
	return support.OfficialConfirmation &&
		support.ErrorDocCount == 0 &&
		support.TruncatedDocCount == 0 &&
		support.TruncatedSnippetCount == 0 &&
		support.UnknownDocCount == 0
}

func reviewExternalDocSafeForAbsorbedPrompt(doc ReviewExternalDocEvidence) bool {
	return strings.TrimSpace(doc.DocID) != "" &&
		strings.TrimSpace(doc.URL) != "" &&
		!doc.FetchedAt.IsZero() &&
		strings.TrimSpace(doc.ContentHash) != "" &&
		strings.TrimSpace(doc.Error) == "" &&
		!doc.Truncated &&
		doc.SourceCredibility == ReviewExternalDocSourceCredibilityOfficialCandidate
}

func reviewExternalDocSnippetSafeForAbsorbedPrompt(snippet ReviewExternalDocSnippetEvidence) bool {
	return strings.TrimSpace(snippet.SnippetID) != "" &&
		strings.TrimSpace(snippet.Content) != "" &&
		strings.TrimSpace(snippet.ContentHash) != "" &&
		!snippet.Truncated
}

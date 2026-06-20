package promptreduction

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	"github.com/susugadx/xelyon-cli/internal/token"
)

// CompactReviewWebSearchDiscoveryEvidence は citation-capable external docs が揃った場合に discovery-only snippets を置換する。
func CompactReviewWebSearchDiscoveryEvidence(bundle ReviewEvidenceBundle) (ReviewEvidenceBundle, int, int, bool) {
	evidence := bundle.WebSearchEvidence
	if !evidence.Enabled ||
		strings.TrimSpace(evidence.Error) != "" ||
		evidence.Truncated ||
		evidence.Inconclusive ||
		len(evidence.ExternalDocs) == 0 {
		return ReviewEvidenceBundle{}, 0, 0, false
	}
	for _, query := range evidence.Queries {
		if strings.TrimSpace(query.Error) != "" {
			return ReviewEvidenceBundle{}, 0, 0, false
		}
	}
	support := externaldoc.SummarizeExternalSupport(evidence)
	if !support.OfficialConfirmation ||
		support.ErrorDocCount > 0 ||
		support.TruncatedDocCount > 0 ||
		support.TruncatedSnippetCount > 0 ||
		support.UnknownDocCount > 0 {
		return ReviewEvidenceBundle{}, 0, 0, false
	}

	compacted := cloneReviewEvidenceBundleForPromptCompact(bundle)
	originalBytes := 0
	replacementBytes := 0
	changed := false
	for queryIndex := range compacted.WebSearchEvidence.Queries {
		for resultIndex := range compacted.WebSearchEvidence.Queries[queryIndex].Results {
			result := &compacted.WebSearchEvidence.Queries[queryIndex].Results[resultIndex]
			snippet := strings.TrimSpace(result.Snippet)
			if snippet == "" || strings.TrimSpace(result.URL) == "" {
				continue
			}
			replacement := reviewWebSearchDiscoverySnippetPlaceholder(*result)
			if len(replacement) >= len(result.Snippet) {
				continue
			}
			originalBytes += len(result.Snippet)
			replacementBytes += len(replacement)
			result.Snippet = replacement
			changed = true
		}
	}
	if !changed {
		return ReviewEvidenceBundle{}, 0, 0, false
	}
	savedBytes := originalBytes - replacementBytes
	savedTokens := token.EstimateTokenCount(strings.Repeat("x", originalBytes)) - token.EstimateTokenCount(strings.Repeat("x", replacementBytes))
	if savedBytes <= 0 || savedTokens < reviewWebSearchDiscoveryCompactMinSavedTokens {
		return ReviewEvidenceBundle{}, 0, 0, false
	}
	return compacted, savedBytes, savedTokens, true
}

func reviewWebSearchDiscoverySnippetPlaceholder(result ReviewWebSearchEvidenceResult) string {
	return fmt.Sprintf(
		"[compacted discovery-only web_search snippet; url=%s; source_domain=%s; snippet_hash=%s; raw_result_preserved=review_artifact]",
		oneLine(result.URL),
		oneLine(result.SourceDomain),
		ReviewPromptShortHash(result.Snippet),
	)
}

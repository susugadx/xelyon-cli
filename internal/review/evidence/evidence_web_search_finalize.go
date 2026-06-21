package evidence

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func finalizeReviewWebSearchEvidenceWithError(evidence externaldoc.WebSearchEvidence, message string) externaldoc.WebSearchEvidence {
	evidence.Error = appendReviewWebSearchEvidenceError(evidence.Error, message)
	return finalizeReviewWebSearchEvidence(evidence)
}

func finalizeReviewWebSearchEvidence(evidence externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	evidence.Inconclusive = !externaldoc.HasFetchedSnippet(evidence.ExternalDocs)
	return evidence
}

func appendReviewWebSearchEvidenceError(existing, message string) string {
	existing = strings.TrimSpace(existing)
	message = strings.TrimSpace(message)
	switch {
	case existing == "":
		return message
	case message == "":
		return existing
	default:
		return existing + "; " + message
	}
}

func cloneReviewWebSearchEvidence(evidence externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	clone := evidence
	clone.Queries = append([]externaldoc.WebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]externaldoc.WebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]externaldoc.Evidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]externaldoc.SnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

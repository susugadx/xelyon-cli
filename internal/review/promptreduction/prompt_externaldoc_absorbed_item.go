package promptreduction

import (
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func reviewExternalDocAbsorbedSnippetPlaceholder(doc externaldoc.Evidence, snippet externaldoc.SnippetEvidence, usedFor []string) string {
	return fmt.Sprintf(
		"[compacted absorbed external_doc snippet; doc_id=%s; snippet_id=%s; url=%s; source_domain=%s; source_credibility=%s; fetched_at=%s; content_hash=%s; used_for=%s; external_support.official_confirmation=true; raw_artifact_ref=%s]",
		oneLine(doc.DocID),
		oneLine(snippet.SnippetID),
		oneLine(doc.URL),
		oneLine(doc.SourceDomain),
		oneLine(string(doc.SourceCredibility)),
		doc.FetchedAt.Format(time.RFC3339Nano),
		oneLine(snippet.ContentHash),
		oneLine(strings.Join(usedFor, ",")),
		reviewWebSearchEvidenceRawArtifactRef,
	)
}

func reviewExternalDocAbsorbedPromptReductionItem(phase ReviewModelPhase, doc externaldoc.Evidence, snippet externaldoc.SnippetEvidence, usedFor []string, originalBytes, replacementBytes int) ReviewPromptReductionItem {
	return ReviewPromptReductionItem{
		ID:         "external_doc:" + strings.TrimSpace(doc.DocID) + ":" + strings.TrimSpace(snippet.SnippetID),
		Family:     ReviewPromptReductionFamilyExternalDoc,
		Phase:      phase,
		Status:     ReviewPromptReductionItemAbsorbed,
		AbsorbedBy: ReviewPromptAbsorptionRefsFromOwners(usedFor),
		EvidenceRefs: []reviewreport.ReviewEvidenceRef{{
			Kind:        reviewreport.ReviewEvidenceKindExternalDoc,
			DocID:       strings.TrimSpace(doc.DocID),
			SnippetID:   strings.TrimSpace(snippet.SnippetID),
			URL:         strings.TrimSpace(doc.URL),
			FetchedAt:   doc.FetchedAt.Format(time.RFC3339Nano),
			ContentHash: strings.TrimSpace(snippet.ContentHash),
		}},
		RawArtifactRef:   reviewWebSearchEvidenceRawArtifactRef,
		Summary:          fmt.Sprintf("external_doc snippet %q is absorbed by latest report scope coverage; raw web evidence remains in %s", strings.TrimSpace(snippet.SnippetID), reviewWebSearchEvidenceRawArtifactRef),
		OriginalBytes:    originalBytes,
		ReplacementBytes: replacementBytes,
	}
}

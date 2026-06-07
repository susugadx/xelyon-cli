package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const reviewWebSearchDiscoveryCompactMinSavedTokens = 128
const reviewExternalDocAbsorbedCompactMinSavedTokens = 128
const reviewWebSearchEvidenceRawArtifactRef = "web_search_evidence*.json"

type reviewExternalDocAbsorptionRefSummary struct {
	owners        []string
	urls          map[string]struct{}
	fetchedAt     map[string]struct{}
	contentHashes map[string]struct{}
}

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
	if compacted, _, _, ok := compactReviewWebSearchDiscoveryEvidence(bundle); ok {
		baseBundle = compacted
	}
	compactedBundle, items, savedBytes, savedTokens, ok := compactReviewExternalDocAbsorbedEvidence(phase, baseBundle, report)
	if !ok {
		return rawMarkdown
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(r.promptReductionMode)
	}
	applied := mode == ReviewPromptReductionModeApply
	r.promptReductionStats.record("review_external_doc_absorbed", savedBytes, savedTokens, applied)
	for _, item := range items {
		r.recordPromptReductionItem(item)
	}
	if !applied {
		return rawMarkdown
	}
	return RenderReviewEvidenceMarkdown(compactedBundle)
}

func compactReviewWebSearchDiscoveryEvidence(bundle ReviewEvidenceBundle) (ReviewEvidenceBundle, int, int, bool) {
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
		reviewPromptShortHash(result.Snippet),
	)
}

func compactReviewExternalDocAbsorbedEvidence(phase ReviewModelPhase, bundle ReviewEvidenceBundle, report ReviewReport) (ReviewEvidenceBundle, []ReviewPromptReductionItem, int, int, bool) {
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

func reviewExternalDocAbsorptionRefs(report ReviewReport) (map[string]reviewExternalDocAbsorptionRefSummary, map[string]struct{}) {
	safeRefs := make(map[string]reviewExternalDocAbsorptionRefSummary)
	unsafeRefs := make(map[string]struct{})
	addRefs := func(refs []ReviewEvidenceRef, owner string, safe bool) {
		for _, ref := range refs {
			if ref.Kind != ReviewEvidenceKindExternalDoc {
				continue
			}
			key := reviewExternalDocSnippetAbsorptionKey(ref.DocID, ref.SnippetID)
			if key == "" {
				continue
			}
			if safe && strings.TrimSpace(ref.URL) != "" && strings.TrimSpace(ref.FetchedAt) != "" && strings.TrimSpace(ref.ContentHash) != "" {
				summary := safeRefs[key]
				summary.owners = append(summary.owners, owner)
				if summary.urls == nil {
					summary.urls = make(map[string]struct{})
				}
				if summary.fetchedAt == nil {
					summary.fetchedAt = make(map[string]struct{})
				}
				if summary.contentHashes == nil {
					summary.contentHashes = make(map[string]struct{})
				}
				summary.urls[strings.TrimSpace(ref.URL)] = struct{}{}
				summary.fetchedAt[strings.TrimSpace(ref.FetchedAt)] = struct{}{}
				summary.contentHashes[strings.TrimSpace(ref.ContentHash)] = struct{}{}
				safeRefs[key] = summary
				continue
			}
			unsafeRefs[key] = struct{}{}
		}
	}

	if report.ScopeCoverage != nil {
		for _, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
			owner := "scope_coverage.surface." + strings.TrimSpace(surface.SurfaceID)
			addRefs(surface.EvidenceRefs, owner, surface.Status == ReviewReportImpactSurfaceChecked)
		}
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			owner := "scope_coverage.risk." + strings.TrimSpace(risk.RiskID)
			addRefs(risk.EvidenceRefs, owner, risk.Status == ReviewReportCandidateRiskDismissed)
		}
		for _, finding := range report.ScopeCoverage.NewFindingsFromReportPass {
			addRefs(finding.EvidenceRefs, "scope_coverage.new_finding", false)
		}
	}

	for _, surface := range report.CheckedSurfaces {
		addRefs(surface.EvidenceRefs, "checked_surfaces", false)
	}
	for _, surface := range report.UnverifiedSurfaces {
		addRefs(surface.EvidenceRefs, "unverified_surfaces", false)
	}
	for _, risk := range report.ResidualRisks {
		addRefs(risk.EvidenceRefs, "residual_risks", false)
	}
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			addRefs(finding.EvidenceRefs, "root_cause.finding", false)
			for _, surface := range finding.CheckedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.checked_surface", false)
			}
			for _, surface := range finding.UnverifiedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.unverified_surface", false)
			}
			for _, risk := range finding.ResidualRisks {
				addRefs(risk.EvidenceRefs, "root_cause.finding.residual_risk", false)
			}
		}
		for _, surface := range group.CheckedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.checked_surface", false)
		}
		for _, surface := range group.UnverifiedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.unverified_surface", false)
		}
		for _, risk := range group.ResidualRisks {
			addRefs(risk.EvidenceRefs, "root_cause.residual_risk", false)
		}
	}

	for key, refs := range safeRefs {
		refs.owners = dedupeSortedReviewPromptAbsorptionRefs(refs.owners)
		safeRefs[key] = refs
	}
	return safeRefs, unsafeRefs
}

func (s reviewExternalDocAbsorptionRefSummary) matches(doc ReviewExternalDocEvidence, snippet ReviewExternalDocSnippetEvidence) bool {
	if len(s.urls) == 0 || len(s.fetchedAt) == 0 || len(s.contentHashes) == 0 {
		return false
	}
	if _, ok := s.urls[strings.TrimSpace(doc.URL)]; !ok {
		return false
	}
	if _, ok := s.fetchedAt[doc.FetchedAt.Format(time.RFC3339Nano)]; !ok {
		return false
	}
	if _, ok := s.contentHashes[strings.TrimSpace(snippet.ContentHash)]; !ok {
		return false
	}
	return true
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

func reviewExternalDocAbsorbedSnippetPlaceholder(doc ReviewExternalDocEvidence, snippet ReviewExternalDocSnippetEvidence, usedFor []string) string {
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

func reviewExternalDocAbsorbedPromptReductionItem(phase ReviewModelPhase, doc ReviewExternalDocEvidence, snippet ReviewExternalDocSnippetEvidence, usedFor []string, originalBytes, replacementBytes int) ReviewPromptReductionItem {
	return ReviewPromptReductionItem{
		ID:         "external_doc:" + strings.TrimSpace(doc.DocID) + ":" + strings.TrimSpace(snippet.SnippetID),
		Family:     ReviewPromptReductionFamilyExternalDoc,
		Phase:      phase,
		Status:     ReviewPromptReductionItemAbsorbed,
		AbsorbedBy: reviewPromptAbsorptionRefsFromOwners(usedFor),
		EvidenceRefs: []ReviewEvidenceRef{{
			Kind:        ReviewEvidenceKindExternalDoc,
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

func reviewExternalDocSnippetAbsorptionKey(docID, snippetID string) string {
	docID = strings.TrimSpace(docID)
	snippetID = strings.TrimSpace(snippetID)
	if docID == "" || snippetID == "" {
		return ""
	}
	return docID + "\x00" + snippetID
}

func cloneReviewEvidenceBundleForPromptCompact(bundle ReviewEvidenceBundle) ReviewEvidenceBundle {
	clone := bundle
	clone.ChangedFiles = append([]ReviewChangedFile(nil), bundle.ChangedFiles...)
	clone.ChangedFileContext = append([]ReviewContextFileEvidence(nil), bundle.ChangedFileContext...)
	clone.RelatedContextFiles = append([]ReviewContextFileEvidence(nil), bundle.RelatedContextFiles...)
	clone.RelatedSearchHits = append([]ReviewRelatedSearchHit(nil), bundle.RelatedSearchHits...)
	clone.GenericImpactCandidatePaths = append([]string(nil), bundle.GenericImpactCandidatePaths...)
	clone.GenericImpactCandidates.Tokens = append([]string(nil), bundle.GenericImpactCandidates.Tokens...)
	clone.GenericImpactCandidates.Candidates = append([]ReviewGenericImpactCandidate(nil), bundle.GenericImpactCandidates.Candidates...)
	clone.UntrackedFiles = append([]ReviewUntrackedFile(nil), bundle.UntrackedFiles...)
	clone.RuleFiles = append([]ReviewRuleFileEvidence(nil), bundle.RuleFiles...)
	clone.Diffs = append([]ReviewDiffEvidence(nil), bundle.Diffs...)
	clone.Inventory = cloneReviewChangeInventory(bundle.Inventory)
	clone.WebSearchEvidence = cloneReviewWebSearchEvidenceForPromptCompact(bundle.WebSearchEvidence)
	return clone
}

func cloneReviewChangeInventory(inventory ReviewChangeInventory) ReviewChangeInventory {
	return ReviewChangeInventory{
		Generated:    append([]string(nil), inventory.Generated...),
		Tests:        append([]string(nil), inventory.Tests...),
		Docs:         append([]string(nil), inventory.Docs...),
		Config:       append([]string(nil), inventory.Config...),
		Production:   append([]string(nil), inventory.Production...),
		NewFiles:     append([]string(nil), inventory.NewFiles...),
		DeletedFiles: append([]string(nil), inventory.DeletedFiles...),
		RenamedFiles: append([]string(nil), inventory.RenamedFiles...),
		Untracked:    append([]string(nil), inventory.Untracked...),
	}
}

func cloneReviewWebSearchEvidenceForPromptCompact(evidence ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	clone := evidence
	clone.Queries = append([]ReviewWebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]ReviewWebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]ReviewExternalDocEvidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]ReviewExternalDocSnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

func reviewPromptShortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

package analysis

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func reviewPressureSignalWebSearchEvidenceDisabledForExternalContractEvidence(input EvidenceInput) []string {
	if input.WebSearchEvidence.Enabled {
		return nil
	}
	subjects := externaldoc.SearchSubjectsForCorpus(reviewPressureSignalWebSearchCorpus(input))
	if len(subjects) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("external_contract_subject", subjects)
}

func reviewPressureSignalWebSearchEvidencePresentEvidence(input EvidenceInput) []string {
	if !input.WebSearchEvidence.Enabled || !externaldoc.HasFetchedSnippet(input.WebSearchEvidence.ExternalDocs) {
		return nil
	}
	return reviewPressureSignalExternalDocEvidence(input.WebSearchEvidence.ExternalDocs, false)
}

func reviewPressureSignalWebSearchEvidenceFailedEvidence(input EvidenceInput) []string {
	if !input.WebSearchEvidence.Enabled {
		return nil
	}
	evidence := make([]string, 0)
	if input.WebSearchEvidence.Error != "" {
		evidence = append(evidence, "web_search_evidence_error: "+input.WebSearchEvidence.Error)
	}
	for _, query := range input.WebSearchEvidence.Queries {
		if query.Error != "" {
			evidence = append(evidence, "web_search_query_error: "+query.Query)
		}
	}
	for _, doc := range input.WebSearchEvidence.ExternalDocs {
		if doc.Error != "" {
			evidence = append(evidence, "external_doc_error: "+doc.DocID+" "+doc.SourceDomain)
		}
	}
	return reviewPressureSignalDedupeEvidence(evidence)
}

func reviewPressureSignalWebSearchEvidenceTruncatedEvidence(input EvidenceInput) []string {
	if !input.WebSearchEvidence.Enabled || !input.WebSearchEvidence.Truncated && !input.TruncationFlags.WebSearchEvidence {
		return nil
	}
	evidence := []string{"web_search_evidence: truncated"}
	evidence = append(evidence, reviewPressureSignalExternalDocEvidence(input.WebSearchEvidence.ExternalDocs, true)...)
	return evidence
}

func reviewPressureSignalWebSearchEvidenceInconclusiveEvidence(input EvidenceInput) []string {
	if !input.WebSearchEvidence.Enabled || !input.WebSearchEvidence.Inconclusive {
		return nil
	}
	evidence := []string{"web_search_evidence: inconclusive"}
	if len(input.WebSearchEvidence.Queries) == 0 {
		evidence = append(evidence, "web_search_queries: []")
	}
	if !externaldoc.HasFetchedSnippet(input.WebSearchEvidence.ExternalDocs) {
		evidence = append(evidence, "external_doc_snippets: []")
	}
	return evidence
}

func reviewPressureSignalExternalDocEvidence(docs []externaldoc.Evidence, onlyTruncated bool) []string {
	evidence := make([]string, 0, minReviewAnalysisInt(len(docs), reviewPressureSignalMaxPathEvidence))
	for _, doc := range docs {
		if onlyTruncated && !doc.Truncated {
			continue
		}
		item := "external_doc: " + doc.DocID
		if doc.SourceDomain != "" {
			item += " " + doc.SourceDomain
		}
		if doc.Truncated {
			item += " truncated"
		}
		evidence = append(evidence, item)
		if len(evidence) == reviewPressureSignalMaxPathEvidence {
			break
		}
	}
	return evidence
}

func reviewPressureSignalWebSearchCorpus(input EvidenceInput) string {
	var parts []string
	parts = append(parts, reviewPressureSignalAllInventoryPaths(input.ChangeInventory)...)
	parts = append(parts, input.GenericImpact.Tokens...)
	for _, diff := range input.Diffs {
		parts = append(parts, diff.Stat.Content, diff.NameStatus.Content, diff.Diff.Content)
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

package review

import (
	"path/filepath"
	"strconv"
	"strings"
)

const reviewPressureSignalMaxPathEvidence = 8

type reviewPressureSignalInput struct {
	Signal   string   `json:"signal"`
	Summary  string   `json:"summary"`
	Evidence []string `json:"evidence"`
}

type reviewPressureSignalSpec struct {
	signal   string
	summary  string
	evidence func(ReviewEvidenceModelInput) []string
}

var reviewPressureSignalSpecs = []reviewPressureSignalSpec{
	{
		signal:   "production_changed_without_tests",
		summary:  "Production files changed and the inventory has no test changes.",
		evidence: reviewPressureSignalProductionWithoutTestsEvidence,
	},
	{
		signal:   "config_or_schema_changed",
		summary:  "Config, schema, or contract-like paths changed.",
		evidence: reviewPressureSignalConfigOrSchemaEvidence,
	},
	{
		signal:   "prompt_contract_changed",
		summary:  "Prompt, instruction, or AGENTS-like paths changed.",
		evidence: reviewPressureSignalPromptContractEvidence,
	},
	{
		signal:   "deleted_or_renamed_files",
		summary:  "Deleted or renamed files may leave stale references or command paths.",
		evidence: reviewPressureSignalDeletedOrRenamedEvidence,
	},
	{
		signal:   "untracked_files_present",
		summary:  "Untracked files are part of the review surface.",
		evidence: reviewPressureSignalUntrackedEvidence,
	},
	{
		signal:   "related_context_empty_or_shallow",
		summary:  "Related context is absent, skipped, or truncated.",
		evidence: reviewPressureSignalRelatedContextEvidence,
	},
	{
		signal:   "related_search_empty_or_truncated",
		summary:  "Related search hits are absent or truncated.",
		evidence: reviewPressureSignalRelatedSearchEvidence,
	},
	{
		signal:   "diff_or_context_truncated",
		summary:  "Status, diff, context, rule, or untracked evidence was truncated.",
		evidence: reviewPressureSignalDiffOrContextTruncatedEvidence,
	},
	{
		signal:   "generated_files_changed",
		summary:  "Generated files changed and may need source-of-truth verification.",
		evidence: reviewPressureSignalGeneratedEvidence,
	},
	{
		signal:   "generic_impact_candidates_present",
		summary:  "Generic impact expansion produced review leads.",
		evidence: reviewPressureSignalGenericImpactCandidatesPresentEvidence,
	},
	{
		signal:   "generic_impact_candidates_truncated",
		summary:  "Generic impact expansion hit a budget and may be incomplete.",
		evidence: reviewPressureSignalGenericImpactCandidatesTruncatedEvidence,
	},
	{
		signal:   "generic_impact_candidates_include_tests_or_docs",
		summary:  "Generic impact candidates include tests/specs or docs/readme references.",
		evidence: reviewPressureSignalGenericImpactCandidatesTestsOrDocsEvidence,
	},
	{
		signal:   "generic_impact_candidates_empty_for_non_go_change",
		summary:  "A non-Go change produced no generic impact candidates.",
		evidence: reviewPressureSignalGenericImpactCandidatesEmptyForNonGoEvidence,
	},
	{
		signal:   "web_search_evidence_disabled_for_external_contract_change",
		summary:  "External-contract-like changes are present, but review web search evidence is disabled.",
		evidence: reviewPressureSignalWebSearchEvidenceDisabledForExternalContractEvidence,
	},
	{
		signal:   "web_search_evidence_present",
		summary:  "Review web search evidence has fetched external document snippets.",
		evidence: reviewPressureSignalWebSearchEvidencePresentEvidence,
	},
	{
		signal:   "web_search_evidence_failed",
		summary:  "Review web search evidence hit search or fetch errors.",
		evidence: reviewPressureSignalWebSearchEvidenceFailedEvidence,
	},
	{
		signal:   "web_search_evidence_truncated",
		summary:  "Review web search evidence hit query, result, response, or snippet limits.",
		evidence: reviewPressureSignalWebSearchEvidenceTruncatedEvidence,
	},
	{
		signal:   "web_search_evidence_inconclusive",
		summary:  "Review web search evidence did not produce fetched snippets that can confirm external specs.",
		evidence: reviewPressureSignalWebSearchEvidenceInconclusiveEvidence,
	},
}

func buildReviewPressureSignalInputs(input ReviewEvidenceModelInput) []reviewPressureSignalInput {
	signals := make([]reviewPressureSignalInput, 0)
	for _, spec := range reviewPressureSignalSpecs {
		evidence := spec.evidence(input)
		if len(evidence) == 0 {
			continue
		}
		signals = append(signals, newReviewPressureSignalInput(spec.signal, spec.summary, evidence))
	}
	return signals
}

func newReviewPressureSignalInput(signal, summary string, evidence []string) reviewPressureSignalInput {
	if evidence == nil {
		evidence = []string{}
	}
	return reviewPressureSignalInput{
		Signal:   signal,
		Summary:  summary,
		Evidence: evidence,
	}
}

func reviewPressureSignalProductionWithoutTestsEvidence(input ReviewEvidenceModelInput) []string {
	inventory := input.ChangeInventory
	if len(inventory.Production) == 0 || len(inventory.Tests) > 0 {
		return nil
	}
	return append(reviewPressureSignalPathEvidence("production", inventory.Production), "tests: []")
}

func reviewPressureSignalConfigOrSchemaEvidence(input ReviewEvidenceModelInput) []string {
	inventory := input.ChangeInventory
	evidence := make([]string, 0)
	evidence = append(evidence, reviewPressureSignalPathEvidence("config", inventory.Config)...)
	evidence = append(evidence, reviewPressureSignalTokenPathEvidence("schema_or_contract_path", reviewPressureSignalAllInventoryPaths(inventory), reviewPressureSignalMatchesSchemaOrContractPath)...)
	return evidence
}

func reviewPressureSignalPromptContractEvidence(input ReviewEvidenceModelInput) []string {
	return reviewPressureSignalTokenPathEvidence("prompt_or_instruction_path", reviewPressureSignalAllInventoryPaths(input.ChangeInventory), reviewPressureSignalMatchesPromptContractPath)
}

func reviewPressureSignalDeletedOrRenamedEvidence(input ReviewEvidenceModelInput) []string {
	inventory := input.ChangeInventory
	if len(inventory.DeletedFiles) == 0 && len(inventory.RenamedFiles) == 0 {
		return nil
	}
	evidence := make([]string, 0, len(inventory.DeletedFiles)+len(inventory.RenamedFiles))
	evidence = append(evidence, reviewPressureSignalPathEvidence("deleted_files", inventory.DeletedFiles)...)
	evidence = append(evidence, reviewPressureSignalPathEvidence("renamed_files", inventory.RenamedFiles)...)
	return evidence
}

func reviewPressureSignalUntrackedEvidence(input ReviewEvidenceModelInput) []string {
	if len(input.ChangeInventory.Untracked) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("untracked", input.ChangeInventory.Untracked)
}

func reviewPressureSignalGeneratedEvidence(input ReviewEvidenceModelInput) []string {
	if len(input.ChangeInventory.Generated) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("generated", input.ChangeInventory.Generated)
}

func reviewPressureSignalGenericImpactCandidatesPresentEvidence(input ReviewEvidenceModelInput) []string {
	if len(input.GenericImpact.Candidates) == 0 {
		return nil
	}
	return reviewPressureSignalGenericImpactCandidateEvidence(input.GenericImpact.Candidates)
}

func reviewPressureSignalGenericImpactCandidatesTruncatedEvidence(input ReviewEvidenceModelInput) []string {
	if !input.GenericImpact.Truncated {
		return nil
	}
	evidence := []string{"generic_impact_candidates: truncated"}
	evidence = append(evidence, reviewPressureSignalGenericImpactCandidateEvidence(input.GenericImpact.Candidates)...)
	return evidence
}

func reviewPressureSignalGenericImpactCandidatesTestsOrDocsEvidence(input ReviewEvidenceModelInput) []string {
	evidence := make([]string, 0)
	for _, candidate := range input.GenericImpact.Candidates {
		switch candidate.Role {
		case ReviewGenericImpactRoleSameStemTestOrSpec, ReviewGenericImpactRoleNearbyTestOrTestsDir, ReviewGenericImpactRoleDocsReference:
			evidence = append(evidence, "generic_impact_candidate: "+candidate.Role+" "+candidate.Path)
		}
		if len(evidence) == reviewPressureSignalMaxPathEvidence {
			break
		}
	}
	return evidence
}

func reviewPressureSignalGenericImpactCandidatesEmptyForNonGoEvidence(input ReviewEvidenceModelInput) []string {
	if len(input.GenericImpact.Candidates) > 0 || !reviewPressureSignalHasNonGoChangedPath(input) {
		return nil
	}
	return []string{"generic_impact_candidates: []", "non_go_changed_paths: present"}
}

func reviewPressureSignalWebSearchEvidenceDisabledForExternalContractEvidence(input ReviewEvidenceModelInput) []string {
	if input.WebSearchEvidence.Enabled {
		return nil
	}
	subjects := reviewWebSearchEvidenceExternalSubjects(reviewPressureSignalWebSearchCorpus(input))
	if len(subjects) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("external_contract_subject", subjects)
}

func reviewPressureSignalWebSearchEvidencePresentEvidence(input ReviewEvidenceModelInput) []string {
	if !input.WebSearchEvidence.Enabled || !reviewWebSearchEvidenceHasFetchedSnippet(input.WebSearchEvidence) {
		return nil
	}
	return reviewPressureSignalExternalDocEvidence(input.WebSearchEvidence.ExternalDocs, false)
}

func reviewPressureSignalWebSearchEvidenceFailedEvidence(input ReviewEvidenceModelInput) []string {
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

func reviewPressureSignalWebSearchEvidenceTruncatedEvidence(input ReviewEvidenceModelInput) []string {
	if !input.WebSearchEvidence.Enabled || !input.WebSearchEvidence.Truncated && !input.TruncationFlags.WebSearchEvidence {
		return nil
	}
	evidence := []string{"web_search_evidence: truncated"}
	evidence = append(evidence, reviewPressureSignalExternalDocEvidence(input.WebSearchEvidence.ExternalDocs, true)...)
	return evidence
}

func reviewPressureSignalWebSearchEvidenceInconclusiveEvidence(input ReviewEvidenceModelInput) []string {
	if !input.WebSearchEvidence.Enabled || !input.WebSearchEvidence.Inconclusive {
		return nil
	}
	evidence := []string{"web_search_evidence: inconclusive"}
	if len(input.WebSearchEvidence.Queries) == 0 {
		evidence = append(evidence, "web_search_queries: []")
	}
	if !reviewWebSearchEvidenceHasFetchedSnippet(input.WebSearchEvidence) {
		evidence = append(evidence, "external_doc_snippets: []")
	}
	return evidence
}

func reviewPressureSignalExternalDocEvidence(docs []ReviewExternalDocEvidence, onlyTruncated bool) []string {
	evidence := make([]string, 0, minReviewEvidenceInt(len(docs), reviewPressureSignalMaxPathEvidence))
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

func reviewPressureSignalWebSearchCorpus(input ReviewEvidenceModelInput) string {
	var parts []string
	parts = append(parts, reviewPressureSignalAllInventoryPaths(input.ChangeInventory)...)
	parts = append(parts, input.GenericImpact.Tokens...)
	for _, diff := range input.Diffs {
		parts = append(parts, diff.Stat.Content, diff.NameStatus.Content, diff.Diff.Content)
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

func reviewPressureSignalGenericImpactCandidateEvidence(candidates []ReviewEvidenceGenericImpactCandidateInput) []string {
	evidence := make([]string, 0, minReviewEvidenceInt(len(candidates), reviewPressureSignalMaxPathEvidence)+1)
	for i, candidate := range candidates {
		if i >= reviewPressureSignalMaxPathEvidence {
			evidence = append(evidence, "generic_impact_candidates: ... ("+strconv.Itoa(len(candidates)-i)+" more)")
			break
		}
		item := "generic_impact_candidate: " + candidate.Role + " " + candidate.Path
		if candidate.Token != "" {
			item += " token=" + candidate.Token
		}
		evidence = append(evidence, item)
	}
	return evidence
}

func reviewPressureSignalHasNonGoChangedPath(input ReviewEvidenceModelInput) bool {
	for _, path := range reviewPressureSignalAllInventoryPaths(input.ChangeInventory) {
		normalized := strings.ToLower(filepath.ToSlash(path))
		if strings.TrimSpace(normalized) == "" || normalized == reviewEvidenceOutsideRepoPathDisplay {
			continue
		}
		if filepath.Ext(normalized) != ".go" {
			return true
		}
	}
	return false
}

func reviewPressureSignalAllInventoryPaths(inventory ReviewEvidenceChangeInventoryInput) []string {
	paths := make([]string, 0,
		len(inventory.Generated)+
			len(inventory.Tests)+
			len(inventory.Docs)+
			len(inventory.Config)+
			len(inventory.Production)+
			len(inventory.NewFiles)+
			len(inventory.DeletedFiles)+
			len(inventory.RenamedFiles)+
			len(inventory.Untracked),
	)
	paths = append(paths, inventory.Generated...)
	paths = append(paths, inventory.Tests...)
	paths = append(paths, inventory.Docs...)
	paths = append(paths, inventory.Config...)
	paths = append(paths, inventory.Production...)
	paths = append(paths, inventory.NewFiles...)
	paths = append(paths, inventory.DeletedFiles...)
	paths = append(paths, inventory.RenamedFiles...)
	paths = append(paths, inventory.Untracked...)
	return paths
}

func reviewPressureSignalMatchesSchemaOrContractPath(path string) bool {
	normalized := strings.ToLower(path)
	return strings.Contains(normalized, "schema") ||
		strings.Contains(normalized, "contract")
}

func reviewPressureSignalMatchesPromptContractPath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	return reviewPressureSignalMatchesKnownRuleFilePath(normalized) ||
		strings.Contains(normalized, "prompt") ||
		strings.Contains(normalized, "instruction") ||
		strings.Contains(normalized, "agents")
}

func reviewPressureSignalMatchesKnownRuleFilePath(normalizedPath string) bool {
	for _, rulePath := range reviewEvidenceRuleFilePaths {
		normalizedRulePath := strings.ToLower(filepath.ToSlash(rulePath))
		if normalizedPath == normalizedRulePath || strings.HasSuffix(normalizedPath, "/"+normalizedRulePath) {
			return true
		}
	}
	return false
}

func reviewPressureSignalRelatedContextEvidence(input ReviewEvidenceModelInput) []string {
	evidence := make([]string, 0)
	if len(input.RelatedContextFiles) == 0 {
		evidence = append(evidence, "related_context_files: []")
	} else if reviewPressureSignalAllRelatedContextSkipped(input.RelatedContextFiles) {
		evidence = append(evidence, "related_context_files: all skipped")
	}
	if input.TruncationFlags.RelatedCandidates {
		evidence = append(evidence, "related_candidates: truncated")
	}
	for _, file := range input.RelatedContextFiles {
		if file.Skipped {
			evidence = append(evidence, reviewPressureSignalRelatedContextFileEvidence("related_context_file skipped", file))
		}
		if file.Truncated {
			evidence = append(evidence, "related_context_file truncated: "+file.Path)
		}
	}
	for _, flag := range input.TruncationFlags.RelatedContextFiles {
		if flag.Truncated {
			evidence = append(evidence, "related_context_truncation flag: "+flag.Path)
		}
	}
	return reviewPressureSignalDedupeEvidence(evidence)
}

func reviewPressureSignalAllRelatedContextSkipped(files []ReviewEvidenceContextFileInput) bool {
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if !file.Skipped {
			return false
		}
	}
	return true
}

func reviewPressureSignalRelatedContextFileEvidence(prefix string, file ReviewEvidenceContextFileInput) string {
	if strings.TrimSpace(file.SkipReason) == "" {
		return prefix + ": " + file.Path
	}
	return prefix + ": " + file.Path + " (" + file.SkipReason + ")"
}

func reviewPressureSignalRelatedSearchEvidence(input ReviewEvidenceModelInput) []string {
	evidence := make([]string, 0, 2)
	if len(input.RelatedSearchHits) == 0 {
		evidence = append(evidence, "related_search_hits: []")
	}
	if input.TruncationFlags.RelatedSearch {
		evidence = append(evidence, "related_search: truncated")
	}
	return evidence
}

func reviewPressureSignalDiffOrContextTruncatedEvidence(input ReviewEvidenceModelInput) []string {
	return reviewPressureSignalTruncationEvidence(input.TruncationFlags)
}

func reviewPressureSignalTruncationEvidence(flags ReviewEvidenceTruncationFlagsInput) []string {
	evidence := make([]string, 0)
	if flags.StatusShort {
		evidence = append(evidence, "status_short: truncated")
	}
	for _, diff := range flags.Diffs {
		if diff.Stat {
			evidence = append(evidence, "diff stat truncated: "+diff.Source)
		}
		if diff.NameStatus {
			evidence = append(evidence, "diff name_status truncated: "+diff.Source)
		}
		if diff.Diff {
			evidence = append(evidence, "diff body truncated: "+diff.Source)
		}
	}
	if flags.UntrackedList {
		evidence = append(evidence, "untracked_list: truncated")
	}
	if flags.UntrackedSnapshots {
		evidence = append(evidence, "untracked_snapshots: truncated")
	}
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("untracked_file", flags.UntrackedFiles)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("rule_file", flags.RuleFiles)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("changed_context_file", flags.ChangedFileContext)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("related_context_file", flags.RelatedContextFiles)...)
	return reviewPressureSignalDedupeEvidence(evidence)
}

func reviewPressureSignalPathTruncationEvidence(prefix string, flags []ReviewEvidencePathTruncationInput) []string {
	evidence := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag.Truncated {
			evidence = append(evidence, prefix+" truncated: "+flag.Path)
		}
	}
	return evidence
}

func reviewPressureSignalPathEvidence(prefix string, paths []string) []string {
	evidence := make([]string, 0, minReviewEvidenceInt(len(paths), reviewPressureSignalMaxPathEvidence)+1)
	for i, path := range paths {
		if i >= reviewPressureSignalMaxPathEvidence {
			evidence = append(evidence, prefix+": ... ("+strconv.Itoa(len(paths)-i)+" more)")
			break
		}
		evidence = append(evidence, prefix+": "+path)
	}
	return evidence
}

func reviewPressureSignalTokenPathEvidence(prefix string, paths []string, match func(string) bool) []string {
	evidence := make([]string, 0)
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok || !match(path) {
			continue
		}
		seen[path] = struct{}{}
		evidence = append(evidence, prefix+": "+path)
		if len(evidence) == reviewPressureSignalMaxPathEvidence {
			break
		}
	}
	return evidence
}

func reviewPressureSignalDedupeEvidence(evidence []string) []string {
	result := make([]string, 0, len(evidence))
	seen := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

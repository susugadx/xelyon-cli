package analysis

const reviewPressureSignalMaxPathEvidence = 8

type reviewPressureSignalSpec struct {
	signal   string
	summary  string
	evidence func(EvidenceInput) []string
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

// BuildPressureSignals は evidence input から Pass1 向け pressure signal を deterministic に構築する。
func BuildPressureSignals(input EvidenceInput, opts PressureSignalOptions) []PressureSignal {
	if len(opts.KnownRuleFilePaths) > 0 {
		input.KnownRuleFilePaths = append([]string(nil), opts.KnownRuleFilePaths...)
	}
	signals := make([]PressureSignal, 0)
	for _, spec := range reviewPressureSignalSpecs {
		evidence := spec.evidence(input)
		if len(evidence) == 0 {
			continue
		}
		signals = append(signals, newReviewPressureSignalInput(spec.signal, spec.summary, evidence))
	}
	return signals
}

func newReviewPressureSignalInput(signal, summary string, evidence []string) PressureSignal {
	if evidence == nil {
		evidence = []string{}
	}
	return PressureSignal{
		Signal:   signal,
		Summary:  summary,
		Evidence: evidence,
	}
}

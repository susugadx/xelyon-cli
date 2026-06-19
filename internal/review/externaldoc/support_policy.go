package externaldoc

func finalizeExternalSupportSummary(summary ExternalSupportSummary) ExternalSupportSummary {
	summary.OfficialConfirmation = summary.Level == ExternalSupportLevelAdequate || summary.Level == ExternalSupportLevelStrong
	if summary.OfficialConfirmation {
		summary.Reasons = append(summary.Reasons, "official_confirmation=true: external support reached adequate or stronger")
	} else {
		summary.Reasons = append(summary.Reasons, "official_confirmation=false: external support is below adequate")
	}
	return summary
}

func externalSupportHasWeakeningSignals(evidence WebSearchEvidence, summary ExternalSupportSummary, hasEvidenceError, hasQueryError bool) bool {
	return evidence.Truncated ||
		evidence.Inconclusive ||
		hasEvidenceError ||
		hasQueryError ||
		summary.ErrorDocCount > 0 ||
		summary.TruncatedDocCount > 0 ||
		summary.TruncatedSnippetCount > 0
}

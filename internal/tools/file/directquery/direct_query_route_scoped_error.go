package directquery

import "strings"

func strictScopedDirectErrorOutcome(input directQueryInput, scopedResolution scopedDirectResolutionOutcome) (Outcome, bool) {
	if !inputHasStrictScopedDirectIntent(input) {
		return Outcome{}, false
	}

	switch scopedResolution.Kind {
	case scopedDirectResolutionMissing:
		return Outcome{
			Kind:  OutcomeError,
			Error: scopedResolution.Error,
		}, true
	case scopedDirectResolutionAmbiguous:
		return Outcome{
			Kind:  OutcomeError,
			Error: "Error: direct path is ambiguous: " + joinDirectQueryRawEntries(input),
		}, true
	default:
		return Outcome{}, false
	}
}

func joinDirectQueryRawEntries(input directQueryInput) string {
	entries := make([]string, 0, len(input.Entries))
	for _, entry := range input.Entries {
		entries = append(entries, entry.RawEntry)
	}
	return strings.Join(entries, ",")
}

func hasScopedExactFilenameLookupScope(policy Policy) bool {
	return policy.ScopedPath != "" || policy.FileFilter != ""
}

func inputHasOnlyScopedDirectCandidates(input directQueryInput) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !entryCanUseScopedDirectResolution(entry) {
			return false
		}
	}
	return true
}

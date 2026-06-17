package file

import "strings"

func strictScopedDirectErrorOutcome(input directQueryInput, scopedResolution scopedDirectResolutionOutcome) (GatherContextDirectRouteOutcome, bool) {
	if !inputHasStrictScopedDirectIntent(input) {
		return GatherContextDirectRouteOutcome{}, false
	}

	switch scopedResolution.Kind {
	case scopedDirectResolutionMissing:
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: scopedResolution.Error,
		}, true
	case scopedDirectResolutionAmbiguous:
		return GatherContextDirectRouteOutcome{
			Kind:  GatherContextDirectRouteOutcomeError,
			Error: "Error: direct path is ambiguous: " + joinDirectQueryRawEntries(input),
		}, true
	default:
		return GatherContextDirectRouteOutcome{}, false
	}
}

func joinDirectQueryRawEntries(input directQueryInput) string {
	entries := make([]string, 0, len(input.Entries))
	for _, entry := range input.Entries {
		entries = append(entries, entry.RawEntry)
	}
	return strings.Join(entries, ",")
}

func hasScopedExactFilenameLookupScope(policy GatherContextDirectRoutePolicy) bool {
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

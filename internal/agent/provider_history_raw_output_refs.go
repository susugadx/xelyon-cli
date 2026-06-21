package agent

import "github.com/susugadx/xelyon-cli/internal/rawoutputs"

func providerHistoryRequiredRawOutputRefs(report ProviderHistoryProjectionReport) ([]rawoutputs.RawOutputRef, []string) {
	refs := providerHistoryAppliedRawOutputRefs(report)
	refs = appendProviderHistoryRequiredRawOutputRefs(refs, report.RawOutputContextRefs)
	return refs, append([]string(nil), report.RawOutputContextMissingRefIDs...)
}

func providerHistoryAppliedRawOutputRefs(report ProviderHistoryProjectionReport) []rawoutputs.RawOutputRef {
	if len(report.RawOutputRefs) == 0 || (len(report.Candidates) == 0 && len(report.CommandEditDryRun.Candidates) == 0) {
		return nil
	}
	refsByID := make(map[string]rawoutputs.RawOutputRef, len(report.RawOutputRefs))
	for _, ref := range report.RawOutputRefs {
		if ref.RefID == "" {
			continue
		}
		refsByID[ref.RefID] = ref
	}
	seen := make(map[string]struct{})
	refs := make([]rawoutputs.RawOutputRef, 0)
	for _, candidate := range report.Candidates {
		if !candidate.ArtifactBackedCandidate || !candidate.ReplacementApplied || candidate.RawOutputRefID == "" {
			continue
		}
		if _, exists := seen[candidate.RawOutputRefID]; exists {
			continue
		}
		ref, ok := refsByID[candidate.RawOutputRefID]
		if !ok {
			continue
		}
		seen[candidate.RawOutputRefID] = struct{}{}
		refs = append(refs, ref)
	}
	for _, candidate := range report.CommandEditDryRun.Candidates {
		if !candidate.ArtifactBackedCandidate || !candidate.ReplacementApplied || candidate.RawOutputRefID == "" {
			continue
		}
		if _, exists := seen[candidate.RawOutputRefID]; exists {
			continue
		}
		ref, ok := refsByID[candidate.RawOutputRefID]
		if !ok {
			continue
		}
		seen[candidate.RawOutputRefID] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

func appendProviderHistoryRequiredRawOutputRefs(first, second []rawoutputs.RawOutputRef) []rawoutputs.RawOutputRef {
	if len(first) == 0 && len(second) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(first)+len(second))
	refs := make([]rawoutputs.RawOutputRef, 0, len(first)+len(second))
	for _, group := range [][]rawoutputs.RawOutputRef{first, second} {
		for _, ref := range group {
			if ref.RefID == "" {
				continue
			}
			if _, ok := seen[ref.RefID]; ok {
				continue
			}
			seen[ref.RefID] = struct{}{}
			refs = append(refs, ref)
		}
	}
	return refs
}

func providerHistoryRawOutputRefIDs(refs []rawoutputs.RawOutputRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.RefID == "" {
			continue
		}
		ids = append(ids, ref.RefID)
	}
	return ids
}

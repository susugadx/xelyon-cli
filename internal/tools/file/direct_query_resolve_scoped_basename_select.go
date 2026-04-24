package file

import "github.com/susugadx/xelyon-cli/internal/filefilter"

func selectScopedBasenameDirectTarget(matches []DirectQueryTarget, fileFilter string, rawEntry string) scopedDirectTargetOutcome {
	if len(matches) == 0 {
		return scopedDirectTargetOutcome{
			Kind:  scopedDirectResolutionMissing,
			Error: "Error: direct path not found: " + rawEntry,
		}
	}

	if fileFilter != "" {
		// Bare/scoped basename resolution is a soft direct route. file_filter
		// must be honored for the final exact target, but a mismatch should not
		// masquerade as a direct read.
		filtered := filterScopedBasenameTargets(matches, fileFilter)
		if len(filtered) == 1 {
			return scopedDirectTargetOutcome{
				Kind:   scopedDirectResolutionResolved,
				Target: filtered[0],
			}
		}
		if len(filtered) == 0 {
			return scopedDirectTargetOutcome{Kind: scopedDirectResolutionFiltered}
		}
		return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
	}

	if len(matches) == 1 {
		return scopedDirectTargetOutcome{
			Kind:   scopedDirectResolutionResolved,
			Target: matches[0],
		}
	}
	return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
}

func filterScopedBasenameTargets(matches []DirectQueryTarget, fileFilter string) []DirectQueryTarget {
	filtered := make([]DirectQueryTarget, 0, len(matches))
	for _, target := range matches {
		if target.Kind != DirectQueryTargetFile {
			filtered = append(filtered, target)
			continue
		}
		if filefilter.Matches(target.FilePath, fileFilter) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

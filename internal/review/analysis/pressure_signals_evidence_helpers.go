package analysis

import "strconv"

func reviewPressureSignalPathEvidence(prefix string, paths []string) []string {
	evidence := make([]string, 0, minReviewAnalysisInt(len(paths), reviewPressureSignalMaxPathEvidence)+1)
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

func minReviewAnalysisInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

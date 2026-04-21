package agent

import (
	"strings"
)

func cleanProjectMapInputPathCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, ".,:;!?()[]{}<>")
	candidate = strings.Trim(candidate, "\"'`")
	candidate = strings.ReplaceAll(candidate, "\\", "/")
	if candidate == "" {
		return ""
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return ""
	}
	if !strings.Contains(candidate, "/") && !strings.Contains(candidate, ".") {
		return ""
	}
	return candidate
}

func filterProjectMapInputCandidates(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}

	type normalizedCandidate struct {
		original   string
		normalized string
	}

	normalized := make([]normalizedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := cleanProjectMapInputPathCandidate(candidate)
		if cleaned == "" {
			continue
		}
		normalized = append(normalized, normalizedCandidate{
			original:   cleaned,
			normalized: strings.ToLower(cleaned),
		})
	}

	filtered := make([]string, 0, len(normalized))
	for i, candidate := range normalized {
		if candidate.original == "" {
			continue
		}
		if strings.Contains(candidate.original, "/") {
			filtered = append(filtered, candidate.original)
			continue
		}

		shadowed := false
		for j, other := range normalized {
			if i == j {
				continue
			}
			if !strings.Contains(other.original, "/") {
				continue
			}
			if strings.HasSuffix(other.normalized, "/"+candidate.normalized) {
				shadowed = true
				break
			}
		}
		if !shadowed {
			filtered = append(filtered, candidate.original)
		}
	}

	return dedupeProjectMapPriorityPaths(filtered)
}

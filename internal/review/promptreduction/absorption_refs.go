package promptreduction

import (
	"sort"
	"strings"
)

// DedupeSortedReviewPromptAbsorptionRefs は absorption owner refs を正規化して安定順にする。
func DedupeSortedReviewPromptAbsorptionRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

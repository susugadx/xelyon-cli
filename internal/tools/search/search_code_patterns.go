package search

import (
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const escapedCommaPlaceholder = "\x00COMMA\x00"

// impactPatternExpansionCap keeps intent=impact conservative and aligned with
// the existing search_code batch-merge size used by the agent optimizer.
const impactPatternExpansionCap = 5

// splitPatterns はカンマ区切りのパターン文字列を分割する。
// \, はリテラルカンマとして扱う。空文字除外、TrimSpace、最大 10 パターン。
func splitPatterns(pattern string) []string {
	s := strings.ReplaceAll(pattern, `\,`, escapedCommaPlaceholder)
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.ReplaceAll(p, escapedCommaPlaceholder, ",")
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// HasEffectivePatternList reports whether a comma-separated search query
// contains at least one non-empty effective pattern after trim and escaped
// comma handling. High-level tools use this to reject malformed separator-only
// input before dispatching into search execution.
func HasEffectivePatternList(pattern string) bool {
	return len(splitPatterns(pattern)) > 0
}

func appendPatternIfMissing(patterns []string, pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return patterns
	}
	for _, existing := range patterns {
		if existing == pattern {
			return patterns
		}
	}
	return append(patterns, pattern)
}

func effectiveSearchPatterns(opts SearchOptions) []string {
	patterns := splitPatterns(opts.Pattern)
	if len(patterns) != 1 {
		return patterns
	}
	if !strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") {
		return patterns
	}
	return expandImpactPatterns(patterns[0], opts)
}

func expandImpactPatterns(pattern string, opts SearchOptions) []string {
	_ = opts
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	variants := []string{
		pattern,
		pattern + "Impl",
	}

	seen := make(map[string]struct{}, len(variants))
	expanded := make([]string, 0, minInt(len(variants), impactPatternExpansionCap))
	for _, candidate := range variants {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		expanded = append(expanded, candidate)
		if len(expanded) >= impactPatternExpansionCap {
			break
		}
	}

	return expanded
}

func shouldExecuteImpactSearch(opts SearchOptions) bool {
	return strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") && len(splitPatterns(opts.Pattern)) == 1
}

func executeImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) string {
	if structured, ok := tryStructuredGoImpactSearch(cache, opts); ok {
		return structured
	}

	basePatterns := expandImpactPatterns(strings.TrimSpace(opts.Pattern), opts)
	if len(basePatterns) == 0 {
		return "Error: pattern is required"
	}

	baseOutput := executeSearchPatterns(cache, basePatterns, opts)
	if impactOutputHasTestCoverage(baseOutput) || len(basePatterns) >= impactPatternExpansionCap {
		return baseOutput
	}

	testProbe := impactTestProbePattern(opts.Pattern)
	if testProbe == "" {
		return baseOutput
	}
	for _, existing := range basePatterns {
		if existing == testProbe {
			return baseOutput
		}
	}

	finalPatterns := append(append([]string(nil), basePatterns...), testProbe)
	return executeSearchPatterns(cache, finalPatterns, opts)
}

func executeSearchPatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	if len(patterns) > 1 {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildMultiSearchCacheKey(opts, patterns)
		if cache != nil {
			if cached, ok := cache.GetSearch(multiKey, cacheKey); ok {
				return cached
			}
		}
		return executeMultiplePatterns(cache, patterns, opts)
	}
	return executeSinglePattern(cache, patterns[0], opts)
}

func impactOutputHasTestCoverage(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "_test.go") || strings.Contains(trimmed, ".test.") || strings.Contains(trimmed, "Tests (") {
			return true
		}
	}
	return false
}

func impactTestProbePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	runes := []rune(pattern)
	runes[0] = unicode.ToUpper(runes[0])
	return "Test" + string(runes)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

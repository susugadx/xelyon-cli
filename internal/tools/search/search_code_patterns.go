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
	if !shouldExpandImpactSearchPatterns(opts, patterns) {
		return patterns
	}
	return expandImpactPatterns(patterns[0], opts)
}

func shouldExpandImpactSearchPatterns(opts SearchOptions, patterns []string) bool {
	return len(patterns) == 1 && strings.EqualFold(strings.TrimSpace(opts.Intent), "impact")
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

	basePatterns := impactBasePatterns(opts)
	if len(basePatterns) == 0 {
		return "Error: pattern is required"
	}

	baseOutput := executeSearchPatterns(cache, basePatterns, opts)
	if !shouldAppendImpactTestProbe(baseOutput, basePatterns) {
		return baseOutput
	}

	finalPatterns := appendImpactTestProbePattern(basePatterns, opts.Pattern)
	if len(finalPatterns) == len(basePatterns) {
		return baseOutput
	}
	return executeSearchPatterns(cache, finalPatterns, opts)
}

func impactBasePatterns(opts SearchOptions) []string {
	return expandImpactPatterns(strings.TrimSpace(opts.Pattern), opts)
}

func shouldAppendImpactTestProbe(baseOutput string, basePatterns []string) bool {
	return !impactOutputHasTestCoverage(baseOutput) && len(basePatterns) < impactPatternExpansionCap
}

func appendImpactTestProbePattern(basePatterns []string, pattern string) []string {
	testProbe := impactTestProbePattern(pattern)
	if testProbe == "" {
		return basePatterns
	}
	for _, existing := range basePatterns {
		if existing == testProbe {
			return basePatterns
		}
	}
	return append(append([]string(nil), basePatterns...), testProbe)
}

func executeSearchPatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	return newSearchPatternDispatch(patterns, opts).execute(cache, opts)
}

type searchPatternDispatch struct {
	patterns []string
	contexts []singlePatternExecutionContext
}

func newSearchPatternDispatch(patterns []string, opts SearchOptions) searchPatternDispatch {
	dispatch := searchPatternDispatch{
		patterns: patterns,
	}
	if len(patterns) > 1 {
		dispatch.contexts = newSinglePatternExecutionContexts(patterns, opts)
	}
	return dispatch
}

func (dispatch searchPatternDispatch) execute(cache tools.ToolCacheInterface, opts SearchOptions) string {
	if len(dispatch.patterns) <= 1 {
		return executeSinglePattern(cache, dispatch.patterns[0], opts)
	}
	return executeMultipleSearchPatterns(cache, dispatch.contexts, opts)
}

func executeMultipleSearchPatterns(cache tools.ToolCacheInterface, contexts []singlePatternExecutionContext, opts SearchOptions) string {
	if cached, ok := loadCachedMultiPatternSearch(cache, contexts, opts); ok {
		return cached
	}
	return executeMultiplePatterns(cache, contexts, opts)
}

func loadCachedMultiPatternSearch(cache tools.ToolCacheInterface, contexts []singlePatternExecutionContext, opts SearchOptions) (string, bool) {
	if cache == nil {
		return "", false
	}
	return cache.GetSearch(buildMultiCacheKeyFromContexts(contexts), buildMultiSearchCacheKeyFromContexts(opts, contexts))
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

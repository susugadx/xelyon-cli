package search

import (
	"fmt"
	"sort"
	"strings"
)

func buildSearchCacheKey(opts SearchOptions, route searchRouteTrace) string {
	return buildSearchCacheKeyWithRoute(opts, route.cacheSignature())
}

func buildMultiCacheKey(patterns []string) string {
	return strings.Join(sortedStrings(patterns), "|")
}

func buildMultiCacheKeyFromContexts(contexts []singlePatternExecutionContext) string {
	return buildMultiCacheKey(contextPatterns(contexts))
}

func buildSearchCacheKeyWithRoute(opts SearchOptions, routeSignature string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|intent=%s|mode=%s|regex=%t|multiline=%t|hidden=%t|ignored=%t|output=%s|ignore=%s|route=%s",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, strings.TrimSpace(opts.Intent), opts.Mode, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored, opts.OutputMode, opts.ignoreKey, routeSignature)
}

func buildMultiSearchCacheKey(opts SearchOptions, patterns []string) string {
	return buildMultiSearchCacheKeyFromContexts(opts, newSinglePatternExecutionContexts(patterns, opts))
}

func buildMultiSearchCacheKeyFromContexts(opts SearchOptions, contexts []singlePatternExecutionContext) string {
	return buildSearchCacheKeyWithRoute(opts, strings.Join(sortedStrings(contextRouteSignatures(contexts)), ";"))
}

func contextPatterns(contexts []singlePatternExecutionContext) []string {
	patterns := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		patterns = append(patterns, ctx.Pattern)
	}
	return patterns
}

func contextRouteSignatures(contexts []singlePatternExecutionContext) []string {
	signatures := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		signatures = append(signatures, ctx.Route.cacheSignature())
	}
	return signatures
}

func sortedStrings(values []string) []string {
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	return sorted
}

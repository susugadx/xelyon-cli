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
	sorted := make([]string, len(patterns))
	copy(sorted, patterns)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func buildSearchCacheKeyWithRoute(opts SearchOptions, routeSignature string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|intent=%s|mode=%s|regex=%t|multiline=%t|hidden=%t|ignored=%t|output=%s|ignore=%s|route=%s",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, strings.TrimSpace(opts.Intent), opts.Mode, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored, opts.OutputMode, opts.ignoreKey, routeSignature)
}

func buildMultiSearchCacheKey(opts SearchOptions, patterns []string) string {
	signatures := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		signatures = append(signatures, planSearchRoute(pattern, opts).cacheSignature())
	}
	sort.Strings(signatures)
	return buildSearchCacheKeyWithRoute(opts, strings.Join(signatures, ";"))
}

package search

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const impactRuntimeRecentActivityLimit = 32

const (
	impactRuntimeRecentFileScore   = 2
	impactRuntimeRecentSearchScore = 1
)

type recentActivityProvider interface {
	RecentFilePaths(limit int) []string
	RecentSearchAffectedFiles(limit int) []string
}

type recentSearchExcludingProvider interface {
	RecentSearchAffectedFilesExcluding(pattern, cacheKey string, limit int) []string
}

type impactRuntimeRankContext struct {
	pattern              string
	cacheKey             string
	excludeCurrentSearch bool
}

func currentSearchImpactRuntimeRankContext(pattern, cacheKey string) impactRuntimeRankContext {
	return impactRuntimeRankContext{
		pattern:              pattern,
		cacheKey:             cacheKey,
		excludeCurrentSearch: true,
	}
}

func formatImpactBundleForRuntime(bundle *SymbolBundle, fallbackOutput string, opts SearchOptions, cache tools.ToolCacheInterface) (*SymbolBundle, string) {
	return formatImpactBundleForRuntimeWithContext(bundle, fallbackOutput, opts, cache, impactRuntimeRankContext{})
}

func formatImpactBundleForRuntimeWithContext(bundle *SymbolBundle, fallbackOutput string, opts SearchOptions, cache tools.ToolCacheInterface, ctx impactRuntimeRankContext) (*SymbolBundle, string) {
	if bundle == nil || bundle.Impact == nil {
		return bundle, fallbackOutput
	}

	ranked := rankImpactBundleForRuntimeWithContext(bundle, opts, cache, ctx)
	return ranked, formatSymbolBundle(ranked, opts.LocatorRegistry, nil)
}

func rankImpactBundleForRuntime(bundle *SymbolBundle, opts SearchOptions, cache tools.ToolCacheInterface) *SymbolBundle {
	return rankImpactBundleForRuntimeWithContext(bundle, opts, cache, impactRuntimeRankContext{})
}

func rankImpactBundleForRuntimeWithContext(bundle *SymbolBundle, opts SearchOptions, cache tools.ToolCacheInterface, ctx impactRuntimeRankContext) *SymbolBundle {
	cloned := cloneSymbolBundle(bundle)
	if cloned == nil || cloned.Impact == nil || len(cloned.Impact.RecommendedReads) <= 1 {
		return cloned
	}

	provider, ok := cache.(recentActivityProvider)
	if !ok || provider == nil {
		return cloned
	}

	recentFiles := recentActivityPathSet(provider.RecentFilePaths(impactRuntimeRecentActivityLimit))
	recentSearchFiles := recentSearchActivityPathSet(cache, provider, ctx)
	if len(recentFiles) == 0 && len(recentSearchFiles) == 0 {
		return cloned
	}

	type rankedRead struct {
		item  SymbolBundleItem
		score int
	}

	ranked := make([]rankedRead, 0, len(cloned.Impact.RecommendedReads)-1)
	for _, item := range cloned.Impact.RecommendedReads[1:] {
		score := 0
		if resolvedPath := resolveImpactRecommendedReadPath(cloned, item, opts); resolvedPath != "" {
			if _, ok := recentFiles[resolvedPath]; ok {
				score += impactRuntimeRecentFileScore
			}
			if _, ok := recentSearchFiles[resolvedPath]; ok {
				score += impactRuntimeRecentSearchScore
			}
		}
		ranked = append(ranked, rankedRead{
			item:  item,
			score: score,
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	reordered := make([]SymbolBundleItem, 0, len(cloned.Impact.RecommendedReads))
	reordered = append(reordered, cloned.Impact.RecommendedReads[0])
	for _, item := range ranked {
		reordered = append(reordered, item.item)
	}
	cloned.Impact.RecommendedReads = reordered
	return cloned
}

func recentSearchActivityPathSet(cache tools.ToolCacheInterface, provider recentActivityProvider, ctx impactRuntimeRankContext) map[string]struct{} {
	if provider == nil {
		return nil
	}
	if !ctx.excludeCurrentSearch {
		return recentActivityPathSet(provider.RecentSearchAffectedFiles(impactRuntimeRecentActivityLimit))
	}

	if excludingProvider, ok := cache.(recentSearchExcludingProvider); ok && excludingProvider != nil {
		return recentActivityPathSet(excludingProvider.RecentSearchAffectedFilesExcluding(ctx.pattern, ctx.cacheKey, impactRuntimeRecentActivityLimit))
	}

	// Cache-hit reranking only uses recent-search signal when the cache can
	// exclude the current entry exactly. File-level filtering would collapse
	// overlapping older searches into the current entry and produce lossy boosts.
	return nil
}

func recentActivityPathSet(paths []string) map[string]struct{} {
	if len(paths) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if cleaned := cleanResolvedLocatorPath(path); cleaned != "" {
			set[cleaned] = struct{}{}
		}
	}
	return set
}

func resolveImpactRecommendedReadPath(bundle *SymbolBundle, item SymbolBundleItem, opts SearchOptions) string {
	if resolved := cleanResolvedLocatorPath(item.ResolvedPath); resolved != "" {
		return resolved
	}

	rootPath := ""
	if bundle != nil {
		rootPath = bundle.Debug.FileRootPath
	}
	return absoluteAffectedFilePathForBundle(item.File, opts, rootPath)
}

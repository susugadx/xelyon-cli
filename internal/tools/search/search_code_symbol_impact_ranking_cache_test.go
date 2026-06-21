package search

import (
	"strings"
	"testing"
)

func TestExecuteSinglePatternDetailed_CacheHitReRanksImpactBundle(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
		recentFilePaths: []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature())
	storeSinglePatternBundle("Run", cacheKey, bundle)
	storeSinglePatternObservation("Run", cacheKey, observationForSymbolBundle(bundle, opts))
	cache.SetSearch("Run", cacheKey, "stale cached output", collectSymbolBundleAffectedFiles(bundle, opts))

	first := executeSinglePatternDetailed(cache, "Run", opts)
	if !strings.Contains(first.Output, "pkg/other.go:30") {
		t.Fatalf("expected reranked cached output to include pkg/other.go near the front, got:\n%s", first.Output)
	}
	if strings.Contains(first.Output, "stale cached output") {
		t.Fatalf("expected cached string fallback to be bypassed when bundle is available, got:\n%s", first.Output)
	}
	if got := first.Bundle.Impact.RecommendedReads[1].File; got != "pkg/other.go" {
		t.Fatalf("expected runtime-ranked bundle to surface pkg/other.go first after definition, got %s", got)
	}
	assertObservationRecommendedReadsMatchBundle(t, first.Observation, first.Bundle)

	cache.recentFilePaths = []string{paths["caller"]}
	second := executeSinglePatternDetailed(cache, "Run", opts)
	if got := second.Bundle.Impact.RecommendedReads[1].File; got != "pkg/caller.go" {
		t.Fatalf("expected cache hit reranking to reflect updated recent activity, got %s", got)
	}
	assertObservationRecommendedReadsMatchBundle(t, second.Observation, second.Bundle)
	if cache.setCalls != 1 {
		t.Fatalf("expected cache hit reranking to reuse the same cache entry without rewriting it, setCalls=%d", cache.setCalls)
	}
}
func TestTryStructuredGoImpactSearch_CacheHitReRanksRecommendedReads(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
		recentFilePaths: []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature()+"|"+structuredGoImpactRouteTag)
	storeSinglePatternBundle("Run", cacheKey, bundle)
	storeSinglePatternObservation("Run", cacheKey, observationForSymbolBundle(bundle, opts))
	cache.SetSearch("Run", cacheKey, "stale cached output", collectSymbolBundleAffectedFiles(bundle, opts))

	firstResult, ok := tryStructuredGoImpactSearchResult(cache, opts)
	if !ok {
		t.Fatal("expected structured impact cache hit")
	}
	assertRecommendedReadOrder(t, firstResult.Rendered, bundle.Impact.RecommendedReads[0], bundle.Impact.RecommendedReads[4], bundle.Impact.RecommendedReads[1])
	assertObservationRecommendedReadsMatchBundle(t, firstResult.Observation, firstResult.Bundle)

	cache.recentFilePaths = []string{paths["caller"]}
	secondResult, ok := tryStructuredGoImpactSearchResult(cache, opts)
	if !ok {
		t.Fatal("expected structured impact cache hit on second lookup")
	}
	assertRecommendedReadOrder(t, secondResult.Rendered, bundle.Impact.RecommendedReads[0], bundle.Impact.RecommendedReads[1], bundle.Impact.RecommendedReads[4])
	assertObservationRecommendedReadsMatchBundle(t, secondResult.Observation, secondResult.Bundle)

	if cache.setCalls != 1 {
		t.Fatalf("expected structured impact cache hit reranking to reuse the same cache entry, setCalls=%d", cache.setCalls)
	}
}
func TestExecuteSinglePatternDetailed_CacheHitRecentSearchExcludesCurrentEntry(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache:           &testSearchCache{data: make(map[string]string)},
		recentSearchAffectedFiles: []string{paths["caller"], paths["test"], paths["other"], paths["shadowResolved"]},
		recentSearchExcluding:     []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature())
	storeSinglePatternBundle("Run", cacheKey, bundle)
	currentAffected := collectSymbolBundleAffectedFiles(bundle, opts)
	cache.SetSearch("Run", cacheKey, "stale cached output", currentAffected)

	result := executeSinglePatternDetailed(cache, "Run", opts)
	if got := result.Bundle.Impact.RecommendedReads[1].File; got != "pkg/other.go" {
		t.Fatalf("expected recent search reranking to exclude current cache entry and surface pkg/other.go, got %s", got)
	}
	if cache.excludingPattern != "Run" || cache.excludingCacheKey != cacheKey {
		t.Fatalf("expected exclusion provider to receive current search identity, got pattern=%q cacheKey=%q", cache.excludingPattern, cache.excludingCacheKey)
	}
}

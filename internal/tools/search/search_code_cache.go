package search

import (
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/searchcache"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type singlePatternExecution struct {
	Pattern       string
	Output        string
	Route         searchRouteTrace
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
}

var singlePatternBundleCache sync.Map
var singlePatternAffectedFilesCache sync.Map
var singlePatternObservationCache sync.Map
var multiPatternObservationCache sync.Map

func init() {
	searchcache.RegisterSearchCacheLifecycleHooks(clearSinglePatternBundleCache, invalidateSinglePatternBundleCacheKeys, invalidateSinglePatternBundleCacheKeys)
}

func singlePatternBundleCacheKey(pattern, cacheKey string) string {
	return pattern + "::" + cacheKey
}

func clearSinglePatternBundleCache() {
	singlePatternBundleCache.Range(func(key, value any) bool {
		singlePatternBundleCache.Delete(key)
		return true
	})
	singlePatternAffectedFilesCache.Range(func(key, value any) bool {
		singlePatternAffectedFilesCache.Delete(key)
		return true
	})
	singlePatternObservationCache.Range(func(key, value any) bool {
		singlePatternObservationCache.Delete(key)
		return true
	})
	multiPatternObservationCache.Range(func(key, value any) bool {
		multiPatternObservationCache.Delete(key)
		return true
	})
}

func invalidateSinglePatternBundleCacheKeys(keys []string) {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		singlePatternBundleCache.Delete(key)
		singlePatternAffectedFilesCache.Delete(key)
		singlePatternObservationCache.Delete(key)
		multiPatternObservationCache.Delete(key)
	}
}

func loadSinglePatternBundle(pattern, cacheKey string) *SymbolBundle {
	value, ok := singlePatternBundleCache.Load(singlePatternBundleCacheKey(pattern, cacheKey))
	if !ok {
		return nil
	}
	bundle, _ := value.(*SymbolBundle)
	return cloneSymbolBundle(bundle)
}

func storeSinglePatternBundle(pattern, cacheKey string, bundle *SymbolBundle) {
	if bundle == nil {
		return
	}
	singlePatternBundleCache.Store(singlePatternBundleCacheKey(pattern, cacheKey), cloneSymbolBundle(bundle))
}

func loadSinglePatternAffectedFiles(pattern, cacheKey string) []string {
	value, ok := singlePatternAffectedFilesCache.Load(singlePatternBundleCacheKey(pattern, cacheKey))
	if !ok {
		return nil
	}
	paths, _ := value.([]string)
	return append([]string(nil), paths...)
}

func storeSinglePatternAffectedFiles(pattern, cacheKey string, affectedFiles []string) {
	if len(affectedFiles) == 0 {
		return
	}
	singlePatternAffectedFilesCache.Store(singlePatternBundleCacheKey(pattern, cacheKey), append([]string(nil), affectedFiles...))
}

func loadSinglePatternObservation(pattern, cacheKey string) *tools.RuntimeObservation {
	value, ok := singlePatternObservationCache.Load(singlePatternBundleCacheKey(pattern, cacheKey))
	if !ok {
		return nil
	}
	observation, _ := value.(*tools.RuntimeObservation)
	return tools.CloneRuntimeObservation(observation)
}

func storeSinglePatternObservation(pattern, cacheKey string, observation *tools.RuntimeObservation) {
	if observation == nil || observation.Empty() {
		return
	}
	singlePatternObservationCache.Store(singlePatternBundleCacheKey(pattern, cacheKey), tools.CloneRuntimeObservation(observation))
}

func loadMultiPatternObservation(patternKey, cacheKey string) *tools.RuntimeObservation {
	value, ok := multiPatternObservationCache.Load(singlePatternBundleCacheKey(patternKey, cacheKey))
	if !ok {
		return nil
	}
	observation, _ := value.(*tools.RuntimeObservation)
	return tools.CloneRuntimeObservation(observation)
}

func storeMultiPatternObservation(patternKey, cacheKey string, observation *tools.RuntimeObservation) {
	if observation == nil || observation.Empty() {
		return
	}
	multiPatternObservationCache.Store(singlePatternBundleCacheKey(patternKey, cacheKey), tools.CloneRuntimeObservation(observation))
}

func cloneSymbolBundle(bundle *SymbolBundle) *SymbolBundle {
	if bundle == nil {
		return nil
	}
	cloned := *bundle
	if bundle.Definition.Body != nil {
		cloned.Definition.Body = append([]string(nil), bundle.Definition.Body...)
	}
	if bundle.Sections != nil {
		cloned.Sections = make([]SymbolBundleSection, len(bundle.Sections))
		for i := range bundle.Sections {
			cloned.Sections[i] = bundle.Sections[i]
			if bundle.Sections[i].Items != nil {
				cloned.Sections[i].Items = append([]SymbolBundleItem(nil), bundle.Sections[i].Items...)
			}
		}
	}
	if bundle.Debug.MatchedPatterns != nil {
		cloned.Debug.MatchedPatterns = append([]string(nil), bundle.Debug.MatchedPatterns...)
	}
	if bundle.Debug.DependencyFiles != nil {
		cloned.Debug.DependencyFiles = append([]string(nil), bundle.Debug.DependencyFiles...)
	}
	cloned.Impact = cloneSymbolBundleImpact(bundle.Impact)
	return &cloned
}

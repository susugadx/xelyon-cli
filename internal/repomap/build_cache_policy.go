package repomap

import "time"

func loadBuildInputCache(rootPath string) *MapCache {
	return loadMapCacheWithFallback(rootPath)
}

func applyCachePolicyToStates(states []fileState, cache *MapCache) []fileState {
	for i := range states {
		cached, ok := resolveReusableCacheFile(cache, states[i].path, states[i].modTime)
		if !ok {
			continue
		}
		states[i].cached = cached
	}
	return states
}

func resolveReusableCacheFile(cache *MapCache, path string, modTime time.Time) (*CacheFile, bool) {
	if cache == nil || cache.Files == nil {
		return nil, false
	}
	cached, ok := cache.Files[path]
	if !ok || cached == nil || !modTime.Equal(cached.ModTime) {
		return nil, false
	}
	return cloneCacheFile(cached), true
}

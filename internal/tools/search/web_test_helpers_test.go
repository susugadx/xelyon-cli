package search

func resetWebSearchCacheForTest() {
	webSearchCacheMu.Lock()
	defer webSearchCacheMu.Unlock()
	webSearchCache = nil
	webSearchCacheSettings = webSearchCacheConfig{}
}

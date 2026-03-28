package tools

import "sync"

var searchCacheLifecycleHooks struct {
	mu         sync.RWMutex
	clear      func()
	invalidate func(keys []string)
	evicted    func(keys []string)
}

// RegisterSearchCacheLifecycleHooks は検索キャッシュの lifecycle hook を登録する。
func RegisterSearchCacheLifecycleHooks(clear func(), invalidate func(keys []string), evicted func(keys []string)) {
	searchCacheLifecycleHooks.mu.Lock()
	defer searchCacheLifecycleHooks.mu.Unlock()
	searchCacheLifecycleHooks.clear = clear
	searchCacheLifecycleHooks.invalidate = invalidate
	searchCacheLifecycleHooks.evicted = evicted
}

// NotifySearchCacheCleared は検索キャッシュ全体クリア後の hook を実行する。
func NotifySearchCacheCleared() {
	searchCacheLifecycleHooks.mu.RLock()
	clear := searchCacheLifecycleHooks.clear
	searchCacheLifecycleHooks.mu.RUnlock()
	if clear != nil {
		clear()
	}
}

// NotifySearchCacheInvalidatedKeys は削除された検索キャッシュ key 群の hook を実行する。
func NotifySearchCacheInvalidatedKeys(keys []string) {
	searchCacheLifecycleHooks.mu.RLock()
	invalidate := searchCacheLifecycleHooks.invalidate
	searchCacheLifecycleHooks.mu.RUnlock()
	if invalidate != nil {
		invalidate(keys)
	}
}

// NotifySearchCacheEvicted は検索キャッシュが容量超過で evict されたときの hook を実行する。
func NotifySearchCacheEvicted(keys []string) {
	searchCacheLifecycleHooks.mu.RLock()
	evicted := searchCacheLifecycleHooks.evicted
	searchCacheLifecycleHooks.mu.RUnlock()
	if evicted != nil {
		evicted(keys)
	}
}

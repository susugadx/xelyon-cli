package searchcache

import "sync"

var searchCacheLifecycleHooks struct {
	mu              sync.RWMutex
	clear           func()
	invalidate      func(keys []string)
	evicted         func(keys []string)
	extraClear      []func()
	extraInvalidate []func(keys []string)
	extraEvicted    []func(keys []string)
}

// RegisterSearchCacheLifecycleHooks は検索キャッシュの lifecycle hook を登録する。
func RegisterSearchCacheLifecycleHooks(clear func(), invalidate func(keys []string), evicted func(keys []string)) {
	searchCacheLifecycleHooks.mu.Lock()
	defer searchCacheLifecycleHooks.mu.Unlock()
	searchCacheLifecycleHooks.clear = clear
	searchCacheLifecycleHooks.invalidate = invalidate
	searchCacheLifecycleHooks.evicted = evicted
}

// AddSearchCacheLifecycleHooks は追加の lifecycle hook を登録する。
func AddSearchCacheLifecycleHooks(clear func(), invalidate func(keys []string), evicted func(keys []string)) {
	searchCacheLifecycleHooks.mu.Lock()
	defer searchCacheLifecycleHooks.mu.Unlock()
	if clear != nil {
		searchCacheLifecycleHooks.extraClear = append(searchCacheLifecycleHooks.extraClear, clear)
	}
	if invalidate != nil {
		searchCacheLifecycleHooks.extraInvalidate = append(searchCacheLifecycleHooks.extraInvalidate, invalidate)
	}
	if evicted != nil {
		searchCacheLifecycleHooks.extraEvicted = append(searchCacheLifecycleHooks.extraEvicted, evicted)
	}
}

// NotifySearchCacheCleared は検索キャッシュ全体クリア後の hook を実行する。
func NotifySearchCacheCleared() {
	searchCacheLifecycleHooks.mu.RLock()
	clear := searchCacheLifecycleHooks.clear
	extraClear := append([]func(){}, searchCacheLifecycleHooks.extraClear...)
	searchCacheLifecycleHooks.mu.RUnlock()
	if clear != nil {
		clear()
	}
	for _, hook := range extraClear {
		if hook != nil {
			hook()
		}
	}
}

// NotifySearchCacheInvalidatedKeys は削除された検索キャッシュ key 群の hook を実行する。
func NotifySearchCacheInvalidatedKeys(keys []string) {
	searchCacheLifecycleHooks.mu.RLock()
	invalidate := searchCacheLifecycleHooks.invalidate
	extraInvalidate := append([]func(keys []string){}, searchCacheLifecycleHooks.extraInvalidate...)
	searchCacheLifecycleHooks.mu.RUnlock()
	if invalidate != nil {
		invalidate(keys)
	}
	for _, hook := range extraInvalidate {
		if hook != nil {
			hook(keys)
		}
	}
}

// NotifySearchCacheEvicted は検索キャッシュが容量超過で evict されたときの hook を実行する。
func NotifySearchCacheEvicted(keys []string) {
	searchCacheLifecycleHooks.mu.RLock()
	evicted := searchCacheLifecycleHooks.evicted
	extraEvicted := append([]func(keys []string){}, searchCacheLifecycleHooks.extraEvicted...)
	searchCacheLifecycleHooks.mu.RUnlock()
	if evicted != nil {
		evicted(keys)
	}
	for _, hook := range extraEvicted {
		if hook != nil {
			hook(keys)
		}
	}
}

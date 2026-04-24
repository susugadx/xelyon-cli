package searchcache

import (
	"slices"
	"strings"
	"testing"
)

func TestLifecycleHooks_NotifiesPrimaryAndExtraHooks(t *testing.T) {
	searchCacheLifecycleHooks.mu.Lock()
	origClear := searchCacheLifecycleHooks.clear
	origInvalidate := searchCacheLifecycleHooks.invalidate
	origEvicted := searchCacheLifecycleHooks.evicted
	origExtraClear := append([]func(){}, searchCacheLifecycleHooks.extraClear...)
	origExtraInvalidate := append([]func([]string){}, searchCacheLifecycleHooks.extraInvalidate...)
	origExtraEvicted := append([]func([]string){}, searchCacheLifecycleHooks.extraEvicted...)
	searchCacheLifecycleHooks.mu.Unlock()
	t.Cleanup(func() {
		searchCacheLifecycleHooks.mu.Lock()
		searchCacheLifecycleHooks.clear = origClear
		searchCacheLifecycleHooks.invalidate = origInvalidate
		searchCacheLifecycleHooks.evicted = origEvicted
		searchCacheLifecycleHooks.extraClear = origExtraClear
		searchCacheLifecycleHooks.extraInvalidate = origExtraInvalidate
		searchCacheLifecycleHooks.extraEvicted = origExtraEvicted
		searchCacheLifecycleHooks.mu.Unlock()
	})

	var events []string
	RegisterSearchCacheLifecycleHooks(
		func() { events = append(events, "clear:primary") },
		func(keys []string) { events = append(events, "invalidate:"+strings.Join(keys, ",")) },
		func(keys []string) { events = append(events, "evict:"+strings.Join(keys, ",")) },
	)
	AddSearchCacheLifecycleHooks(
		func() { events = append(events, "clear:extra") },
		func(keys []string) { events = append(events, "invalidate-extra:"+strings.Join(keys, ",")) },
		nil,
	)
	AddSearchCacheLifecycleHooks(nil, nil, func(keys []string) {
		events = append(events, "evict-extra:"+strings.Join(keys, ","))
	})

	NotifySearchCacheCleared()
	NotifySearchCacheInvalidatedKeys([]string{"a", "b"})
	NotifySearchCacheEvicted([]string{"c"})

	want := []string{
		"clear:primary",
		"clear:extra",
		"invalidate:a,b",
		"invalidate-extra:a,b",
		"evict:c",
		"evict-extra:c",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("hook events = %v, want %v", events, want)
	}
}

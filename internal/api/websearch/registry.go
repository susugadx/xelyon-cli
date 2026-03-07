package websearch

import (
	"fmt"
	"strings"
	"sync"
)

// SearchFunc はプロバイダー固有のネイティブ検索実装。
type SearchFunc func(query, model string) (string, error)

var (
	registry   = make(map[string]SearchFunc)
	registryMu sync.RWMutex
)

// Register registers a provider-native web search implementation.
func Register(providerName string, fn SearchFunc) {
	if fn == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(strings.TrimSpace(providerName))] = fn
}

// Search executes a registered provider-native web search implementation.
func Search(providerName, query, model string) (string, error) {
	registryMu.RLock()
	fn, ok := registry[strings.ToLower(strings.TrimSpace(providerName))]
	registryMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("native web search is not registered for provider %q", providerName)
	}
	return fn(query, model)
}

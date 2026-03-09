package websearch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// SearchFunc はプロバイダー固有のネイティブ検索実装。
type SearchFunc func(query, model string) (string, error)

// SearchFuncWithContext は request context 付きのネイティブ検索実装。
type SearchFuncWithContext func(ctx context.Context, query, model string) (string, error)

var (
	registry   = make(map[string]SearchFuncWithContext)
	registryMu sync.RWMutex
)

// Register registers a provider-native web search implementation.
func Register(providerName string, fn SearchFunc) {
	if fn == nil {
		return
	}
	RegisterWithContext(providerName, func(_ context.Context, query, model string) (string, error) {
		return fn(query, model)
	})
}

// RegisterWithContext は context 対応のプロバイダー固有検索実装を登録する。
func RegisterWithContext(providerName string, fn SearchFuncWithContext) {
	if fn == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(strings.TrimSpace(providerName))] = fn
}

// Search executes a registered provider-native web search implementation.
func Search(providerName, query, model string) (string, error) {
	return SearchWithContext(config.WithContext(context.Background(), config.DefaultConfig()), providerName, query, model)
}

// SearchWithContext は request context を渡してネイティブ検索実装を実行する。
func SearchWithContext(ctx context.Context, providerName, query, model string) (string, error) {
	registryMu.RLock()
	fn, ok := registry[strings.ToLower(strings.TrimSpace(providerName))]
	registryMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("native web search is not registered for provider %q", providerName)
	}
	return fn(ctx, query, model)
}

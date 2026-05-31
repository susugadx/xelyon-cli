package websearch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// SearchFunc はプロバイダー固有のネイティブ検索実装。
type SearchFunc func(query, model string) (string, error)

// SearchFuncWithContext は request context 付きのネイティブ検索実装。
type SearchFuncWithContext func(ctx context.Context, query, model string) (string, error)

var (
	registry   = make(map[string]SearchFuncWithContext)
	registryMu sync.RWMutex
)

type usageCallbackContextKey struct{}

type cleanupRegistrar interface {
	Helper()
	Cleanup(func())
}

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
	registry[normalizeProviderName(providerName)] = fn
}

// RegisterWithContextForTest はテスト用に登録を差し替え、終了時に元の registry 状態を復元する。
func RegisterWithContextForTest(t cleanupRegistrar, providerName string, fn SearchFuncWithContext) {
	if t != nil {
		t.Helper()
	}
	if fn == nil {
		return
	}
	key := normalizeProviderName(providerName)
	registryMu.Lock()
	previous, hadPrevious := registry[key]
	registry[key] = fn
	registryMu.Unlock()
	if t == nil {
		return
	}
	t.Cleanup(func() {
		registryMu.Lock()
		defer registryMu.Unlock()
		if hadPrevious {
			registry[key] = previous
		} else {
			delete(registry, key)
		}
	})
}

// SearchWithContext は request context を渡してネイティブ検索実装を実行する。
func SearchWithContext(ctx context.Context, providerName, query, model string) (string, error) {
	registryMu.RLock()
	fn, ok := registry[normalizeProviderName(providerName)]
	registryMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("native web search is not registered for provider %q", providerName)
	}
	return fn(ctx, query, model)
}

func normalizeProviderName(providerName string) string {
	return strings.ToLower(strings.TrimSpace(providerName))
}

// WithUsageCallback は native web search request context に usage callback を埋め込む。
func WithUsageCallback(ctx context.Context, callback api.UsageCallback) context.Context {
	if callback == nil {
		return ctx
	}
	return context.WithValue(ctx, usageCallbackContextKey{}, callback)
}

// UsageCallbackFromContext は native web search request context から usage callback を取得する。
func UsageCallbackFromContext(ctx context.Context) api.UsageCallback {
	if ctx == nil {
		return nil
	}
	callback, _ := ctx.Value(usageCallbackContextKey{}).(api.UsageCallback)
	return callback
}

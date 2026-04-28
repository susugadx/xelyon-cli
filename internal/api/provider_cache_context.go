package api

import (
	"context"
	"strings"
)

type providerCacheNamespaceKey struct{}

// WithProviderCacheNamespace は provider 内部キャッシュの名前空間を context に関連付ける。
func WithProviderCacheNamespace(ctx context.Context, namespace string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, providerCacheNamespaceKey{}, strings.TrimSpace(namespace))
}

// ProviderCacheNamespaceFromContext は context に設定された provider cache namespace を返す。
func ProviderCacheNamespaceFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if namespace, ok := ctx.Value(providerCacheNamespaceKey{}).(string); ok {
		return strings.TrimSpace(namespace)
	}
	return ""
}

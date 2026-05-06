package api

import (
	"context"
	"strings"
)

type promptCacheScopeContextKey struct{}

// PromptCacheScope は provider が request 単位のプロンプトキャッシュ範囲を決めるための識別子。
type PromptCacheScope struct {
	SessionID string
	TaskID    string
}

// WithPromptCacheScope は provider-neutral なプロンプトキャッシュ scope を context に関連付ける。
func WithPromptCacheScope(ctx context.Context, scope PromptCacheScope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := normalizePromptCacheScope(scope)
	return context.WithValue(ctx, promptCacheScopeContextKey{}, normalized)
}

// PromptCacheScopeFromContext は context に設定されたプロンプトキャッシュ scope を返す。
func PromptCacheScopeFromContext(ctx context.Context) (PromptCacheScope, bool) {
	if ctx == nil {
		return PromptCacheScope{}, false
	}
	scope, ok := ctx.Value(promptCacheScopeContextKey{}).(PromptCacheScope)
	if !ok {
		return PromptCacheScope{}, false
	}
	scope = normalizePromptCacheScope(scope)
	if scope.SessionID == "" && scope.TaskID == "" {
		return PromptCacheScope{}, false
	}
	return scope, true
}

func normalizePromptCacheScope(scope PromptCacheScope) PromptCacheScope {
	return PromptCacheScope{
		SessionID: strings.TrimSpace(scope.SessionID),
		TaskID:    strings.TrimSpace(scope.TaskID),
	}
}

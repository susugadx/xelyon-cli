package api

import (
	"context"
	"sync"
)

type toolDefinitionsContextKey struct{}

var defaultToolDefinitionsState struct {
	mu   sync.RWMutex
	defs []ToolDefinition
}

// SetDefaultToolDefinitions は互換用の既定ツール定義を差し替える。
// 実行時の正規ルートでは request context に明示注入された定義を使用する。
func SetDefaultToolDefinitions(defs []ToolDefinition) {
	defaultToolDefinitionsState.mu.Lock()
	defer defaultToolDefinitionsState.mu.Unlock()
	defaultToolDefinitionsState.defs = cloneToolDefinitions(defs)
}

// WithToolDefinitions は provider が使用するツール定義 surface を request context に注入する。
func WithToolDefinitions(ctx context.Context, defs []ToolDefinition) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, toolDefinitionsContextKey{}, cloneToolDefinitions(defs))
}

// ToolDefinitionsFromContext は provider-neutral なツール定義 surface を返す。
// context に明示注入がない場合は互換用の default surface を返す。
func ToolDefinitionsFromContext(ctx context.Context) []ToolDefinition {
	if ctx != nil {
		if defs, ok := ctx.Value(toolDefinitionsContextKey{}).([]ToolDefinition); ok {
			return cloneToolDefinitions(defs)
		}
	}
	defaultToolDefinitionsState.mu.RLock()
	defer defaultToolDefinitionsState.mu.RUnlock()
	return cloneToolDefinitions(defaultToolDefinitionsState.defs)
}

func cloneToolDefinitions(defs []ToolDefinition) []ToolDefinition {
	if len(defs) == 0 {
		return nil
	}
	cloned := make([]ToolDefinition, len(defs))
	for i, def := range defs {
		cloned[i] = def
		if def.Parameters != nil {
			cloned[i].Parameters = cloneToolDefinitionMap(def.Parameters)
		}
	}
	return cloned
}

func cloneToolDefinitionMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

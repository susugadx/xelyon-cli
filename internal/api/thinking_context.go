package api

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type thinkingOverrideKey struct{}

// WithThinkingDisabled は Thinking を無効化した context を返す
func WithThinkingDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, thinkingOverrideKey{}, false)
}

// IsThinkingEnabled は context のオーバーライドを優先し、
// なければグローバル設定を参照して Thinking が有効かを返す
func IsThinkingEnabled(ctx context.Context) bool {
	if v, ok := ctx.Value(thinkingOverrideKey{}).(bool); ok {
		return v
	}
	cfg := config.GetGlobalConfig()
	return cfg.Thinking.Enabled
}

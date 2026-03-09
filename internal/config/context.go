package config

import "context"

type contextKey struct{}

// WithContext は request context に設定を埋め込む。
func WithContext(ctx context.Context, cfg *Config) context.Context {
	if cfg == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, cfg)
}

// FromContext は request context から設定を取得する。
func FromContext(ctx context.Context) *Config {
	if ctx == nil {
		return DefaultConfig()
	}
	if cfg, ok := ctx.Value(contextKey{}).(*Config); ok && cfg != nil {
		return cfg
	}
	return DefaultConfig()
}

// LookupContext は request context に明示注入された設定のみを取得する。
func LookupContext(ctx context.Context) (*Config, bool) {
	if ctx == nil {
		return nil, false
	}
	cfg, ok := ctx.Value(contextKey{}).(*Config)
	if !ok || cfg == nil {
		return nil, false
	}
	return cfg, true
}

// WithGlobalFallback は context に設定が未注入のとき、現在のグローバル設定を埋め込む。
// グローバル設定が未初期化の場合はデフォルト設定を埋め込む。
func WithGlobalFallback(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := LookupContext(ctx); ok {
		return ctx
	}
	if globalConfig != nil {
		return WithContext(ctx, globalConfig)
	}
	return WithContext(ctx, DefaultConfig())
}

// ResolveContext は request context の設定を優先し、未注入時は fallback、
// それも nil の場合は DefaultConfig を返す。
func ResolveContext(ctx context.Context, fallback *Config) *Config {
	if cfg, ok := LookupContext(ctx); ok {
		return cfg
	}
	if fallback != nil {
		return fallback
	}
	return DefaultConfig()
}

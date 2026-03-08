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
		return GetGlobalConfig()
	}
	if cfg, ok := ctx.Value(contextKey{}).(*Config); ok && cfg != nil {
		return cfg
	}
	return GetGlobalConfig()
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

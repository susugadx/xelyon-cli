package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestIsThinkingEnabled_DefaultFallback(t *testing.T) {
	// Thinking OFF の設定
	cfg := &config.Config{}
	cfg.Thinking.Enabled = false

	ctx := config.WithContext(context.Background(), cfg)
	if IsThinkingEnabled(ctx) {
		t.Error("expected false when config.Thinking.Enabled=false and no override")
	}

	// Thinking ON の設定
	cfgOn := &config.Config{}
	cfgOn.Thinking.Enabled = true
	ctx = config.WithContext(context.Background(), cfgOn)

	if !IsThinkingEnabled(ctx) {
		t.Error("expected true when config.Thinking.Enabled=true and no override")
	}
}

func TestWithThinkingDisabled(t *testing.T) {
	// context 設定は Thinking ON
	cfg := &config.Config{}
	cfg.Thinking.Enabled = true

	// context.Value でオーバーライド
	ctx := WithThinkingDisabled(config.WithContext(context.Background(), cfg))
	if IsThinkingEnabled(ctx) {
		t.Error("expected false after WithThinkingDisabled, even when config.Thinking.Enabled=true")
	}

	// 元の context はオーバーライドなし
	if !IsThinkingEnabled(config.WithContext(context.Background(), cfg)) {
		t.Error("expected true for plain context when config.Thinking.Enabled=true")
	}
}

func TestWithThinkingDisabled_NestedContext(t *testing.T) {
	cfg := &config.Config{}
	cfg.Thinking.Enabled = true

	// WithThinkingDisabled の後に WithTimeout しても override は保持される
	ctx := WithThinkingDisabled(config.WithContext(context.Background(), cfg))
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if IsThinkingEnabled(childCtx) {
		t.Error("expected false for child context derived from WithThinkingDisabled context")
	}
}

func TestIsThinkingEnabled_UsesContextConfig(t *testing.T) {
	ctxCfg := config.DefaultConfig()
	ctxCfg.Thinking.Enabled = true

	ctx := config.WithContext(context.Background(), ctxCfg)
	if !IsThinkingEnabled(ctx) {
		t.Error("expected context config to enable thinking even when global config is disabled")
	}

	if IsThinkingEnabled(WithThinkingDisabled(ctx)) {
		t.Error("expected explicit thinking override to win over context config")
	}
}

package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestIsThinkingEnabled_DefaultFallback(t *testing.T) {
	originalConfig := config.GetGlobalConfig()
	defer config.SetGlobalConfig(originalConfig)

	// Thinking OFF の設定
	cfg := &config.Config{}
	cfg.Thinking.Enabled = false
	config.SetGlobalConfig(cfg)

	ctx := context.Background()
	if IsThinkingEnabled(ctx) {
		t.Error("expected false when config.Thinking.Enabled=false and no override")
	}

	// Thinking ON の設定
	cfg.Thinking.Enabled = true
	config.SetGlobalConfig(cfg)

	if !IsThinkingEnabled(ctx) {
		t.Error("expected true when config.Thinking.Enabled=true and no override")
	}
}

func TestWithThinkingDisabled(t *testing.T) {
	originalConfig := config.GetGlobalConfig()
	defer config.SetGlobalConfig(originalConfig)

	// グローバル設定は Thinking ON
	cfg := &config.Config{}
	cfg.Thinking.Enabled = true
	config.SetGlobalConfig(cfg)

	// context.Value でオーバーライド
	ctx := WithThinkingDisabled(context.Background())
	if IsThinkingEnabled(ctx) {
		t.Error("expected false after WithThinkingDisabled, even when config.Thinking.Enabled=true")
	}

	// 元の context はオーバーライドなし → グローバル設定を参照
	if !IsThinkingEnabled(context.Background()) {
		t.Error("expected true for plain context when config.Thinking.Enabled=true")
	}
}

func TestWithThinkingDisabled_NestedContext(t *testing.T) {
	originalConfig := config.GetGlobalConfig()
	defer config.SetGlobalConfig(originalConfig)

	cfg := &config.Config{}
	cfg.Thinking.Enabled = true
	config.SetGlobalConfig(cfg)

	// WithThinkingDisabled の後に WithTimeout しても override は保持される
	ctx := WithThinkingDisabled(context.Background())
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if IsThinkingEnabled(childCtx) {
		t.Error("expected false for child context derived from WithThinkingDisabled context")
	}
}

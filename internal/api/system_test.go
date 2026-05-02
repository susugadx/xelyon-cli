package api

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildSystemField_SingleBlock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	result := BuildSystemFieldWithConfig("Hello world", cfg)

	blocks, ok := result.([]SystemBlock)
	if !ok {
		t.Fatalf("expected []SystemBlock, got %T", result)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Text != "Hello world" {
		t.Errorf("block text = %q, want 'Hello world'", blocks[0].Text)
	}
	if blocks[0].CacheControl == nil {
		t.Error("expected cache_control on first block")
	}
	if blocks[0].CacheControl.Type != "ephemeral" {
		t.Errorf("cache_control type = %q, want 'ephemeral'", blocks[0].CacheControl.Type)
	}
}

func TestBuildSystemField_SplitBlocks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	prompt := "base prompt" + SystemPromptCacheBoundary + "dynamic part"
	result := BuildSystemFieldWithConfig(prompt, cfg)

	blocks, ok := result.([]SystemBlock)
	if !ok {
		t.Fatalf("expected []SystemBlock, got %T", result)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// First block: static without cache_control
	if blocks[0].Text != "base prompt" {
		t.Errorf("block[0] text = %q, want 'base prompt'", blocks[0].Text)
	}
	if blocks[0].CacheControl != nil {
		t.Error("expected no cache_control on first block (cache_control goes on last block)")
	}

	// Second block: dynamic with cache_control (ensures full prefix is cached)
	if blocks[1].Text != "dynamic part" {
		t.Errorf("block[1] text = %q, want 'dynamic part'", blocks[1].Text)
	}
	if blocks[1].CacheControl == nil {
		t.Error("expected cache_control on last block")
	}
}

func TestBuildSystemField_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false

	prompt := "base prompt" + SystemPromptCacheBoundary + "dynamic part"
	result := BuildSystemFieldWithConfig(prompt, cfg)

	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string when cache disabled, got %T", result)
	}
	if str != "base prompt\n\ndynamic part" {
		t.Errorf("result = %q, want 'base prompt\\n\\ndynamic part'", str)
	}
}

func TestNewCacheControl_DefaultTTL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	// CacheTTL = "5m" (default)

	cc := NewCacheControlWithConfig(cfg)
	if cc.Type != "ephemeral" {
		t.Errorf("type = %q, want 'ephemeral'", cc.Type)
	}
	if cc.TTL != "" {
		t.Errorf("TTL should be empty for default 5m, got %q", cc.TTL)
	}
}

func TestNewCacheControl_ExtendedTTL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	cfg.PromptCache.CacheTTL = "1h"

	cc := NewCacheControlWithConfig(cfg)
	if cc.Type != "ephemeral" {
		t.Errorf("type = %q, want 'ephemeral'", cc.Type)
	}
	if cc.TTL != "1h" {
		t.Errorf("TTL = %q, want '1h'", cc.TTL)
	}
}

func TestNewCacheControlWithConfig_UsesInjectedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.CacheTTL = "1h"

	cc := NewCacheControlWithConfig(cfg)
	if cc.TTL != "1h" {
		t.Fatalf("TTL = %q, want %q", cc.TTL, "1h")
	}
}

func TestBuildSystemField_EmptyDynamic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	prompt := "base prompt" + SystemPromptCacheBoundary + "   "
	result := BuildSystemFieldWithConfig(prompt, cfg)

	blocks, ok := result.([]SystemBlock)
	if !ok {
		t.Fatalf("expected []SystemBlock, got %T", result)
	}
	// Empty dynamic part should result in only 1 block
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (empty dynamic skipped), got %d", len(blocks))
	}
	if blocks[0].Text != "base prompt" {
		t.Errorf("block text = %q, want 'base prompt'", blocks[0].Text)
	}
}

func TestBuildSystemFieldWithConfig_UsesInjectedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	result := BuildSystemFieldWithConfig("Hello world", cfg)

	if _, ok := result.([]SystemBlock); !ok {
		t.Fatalf("expected []SystemBlock with injected config, got %T", result)
	}
}

func TestSplitSystemPromptLayout_MultipleBoundariesAreNormalizedIntoDynamic(t *testing.T) {
	input := "base" + SystemPromptCacheBoundary + "dynamic-a" + SystemPromptCacheBoundary + "dynamic-b"
	layout := SplitSystemPromptLayout(input)
	if layout.Static != "base" {
		t.Fatalf("layout.Static = %q, want base", layout.Static)
	}
	if layout.Dynamic != "dynamic-a\n\ndynamic-b" {
		t.Fatalf("layout.Dynamic = %q, want normalized dynamic", layout.Dynamic)
	}
}

func TestSystemPromptLayout_ComposeWithoutDynamic(t *testing.T) {
	layout := SystemPromptLayout{Static: "base", Dynamic: "   "}
	got := layout.Compose()
	if got != "base" {
		t.Fatalf("Compose() = %q, want base", got)
	}
}

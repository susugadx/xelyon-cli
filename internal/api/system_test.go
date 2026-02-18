package api

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildSystemField_SingleBlock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	result := BuildSystemField("Hello world")

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
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	prompt := "base prompt" + SystemPromptCacheBoundary + "dynamic part"
	result := BuildSystemField(prompt)

	blocks, ok := result.([]SystemBlock)
	if !ok {
		t.Fatalf("expected []SystemBlock, got %T", result)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// First block: static with cache_control
	if blocks[0].Text != "base prompt" {
		t.Errorf("block[0] text = %q, want 'base prompt'", blocks[0].Text)
	}
	if blocks[0].CacheControl == nil {
		t.Error("expected cache_control on first block")
	}

	// Second block: dynamic without cache_control
	if blocks[1].Text != "dynamic part" {
		t.Errorf("block[1] text = %q, want 'dynamic part'", blocks[1].Text)
	}
	if blocks[1].CacheControl != nil {
		t.Error("expected no cache_control on second block")
	}
}

func TestBuildSystemField_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = false
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	prompt := "base prompt" + SystemPromptCacheBoundary + "dynamic part"
	result := BuildSystemField(prompt)

	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string when cache disabled, got %T", result)
	}
	if str != "base prompt\n\ndynamic part" {
		t.Errorf("result = %q, want 'base prompt\\n\\ndynamic part'", str)
	}
}

func TestBuildSystemField_EmptyDynamic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	prompt := "base prompt" + SystemPromptCacheBoundary + "   "
	result := BuildSystemField(prompt)

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

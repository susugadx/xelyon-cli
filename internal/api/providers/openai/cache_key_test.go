package openai

import (
	"strings"
	"testing"
)

func TestBuildPromptCacheKey_Format(t *testing.T) {
	key := buildPromptCacheKeyWithCwd("/home/user/project", "gpt-4o", "You are a helpful assistant")

	parts := strings.Split(key, ":")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts separated by ':', got %d: %q", len(parts), key)
	}
	if parts[0] != "xelyon" {
		t.Errorf("expected prefix 'xelyon', got %q", parts[0])
	}
	if len(parts[1]) != 8 {
		t.Errorf("expected cwd_hash length 8, got %d: %q", len(parts[1]), parts[1])
	}
	if parts[2] != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", parts[2])
	}
	if len(parts[3]) != 8 {
		t.Errorf("expected prompt_hash length 8, got %d: %q", len(parts[3]), parts[3])
	}
}

func TestBuildPromptCacheKey_DifferentCwd(t *testing.T) {
	key1 := buildPromptCacheKeyWithCwd("/home/user/project-a", "gpt-4o", "same prompt")
	key2 := buildPromptCacheKeyWithCwd("/home/user/project-b", "gpt-4o", "same prompt")

	if key1 == key2 {
		t.Errorf("different cwd should produce different keys: %q == %q", key1, key2)
	}
}

func TestBuildPromptCacheKey_DifferentModel(t *testing.T) {
	key1 := buildPromptCacheKeyWithCwd("/same/path", "gpt-4o", "same prompt")
	key2 := buildPromptCacheKeyWithCwd("/same/path", "gpt-4o-mini", "same prompt")

	if key1 == key2 {
		t.Errorf("different model should produce different keys: %q == %q", key1, key2)
	}
}

func TestBuildPromptCacheKey_DifferentPrompt(t *testing.T) {
	key1 := buildPromptCacheKeyWithCwd("/same/path", "gpt-4o", "prompt A")
	key2 := buildPromptCacheKeyWithCwd("/same/path", "gpt-4o", "prompt B")

	if key1 == key2 {
		t.Errorf("different prompts should produce different keys: %q == %q", key1, key2)
	}
}

func TestBuildPromptCacheKey_SameInputsSameKey(t *testing.T) {
	key1 := buildPromptCacheKeyWithCwd("/project", "gpt-4o", "system prompt")
	key2 := buildPromptCacheKeyWithCwd("/project", "gpt-4o", "system prompt")

	if key1 != key2 {
		t.Errorf("same inputs should produce same key: %q != %q", key1, key2)
	}
}

func TestBuildPromptCacheKey_LongPromptTruncated(t *testing.T) {
	longPrompt := strings.Repeat("a", 200)
	// 先頭100文字が同じなら同じキー
	key1 := buildPromptCacheKeyWithCwd("/p", "m", longPrompt)
	key2 := buildPromptCacheKeyWithCwd("/p", "m", longPrompt+"extra suffix")

	if key1 != key2 {
		t.Errorf("prompts with same first 100 chars should produce same key: %q != %q", key1, key2)
	}

	// 先頭100文字が異なれば異なるキー
	differentPrompt := strings.Repeat("b", 100) + strings.Repeat("a", 100)
	key3 := buildPromptCacheKeyWithCwd("/p", "m", differentPrompt)
	if key1 == key3 {
		t.Errorf("prompts with different first 100 chars should produce different keys")
	}
}

func TestShortHash_Deterministic(t *testing.T) {
	h1 := shortHash("test input")
	h2 := shortHash("test input")
	if h1 != h2 {
		t.Errorf("same input should produce same hash: %q != %q", h1, h2)
	}
	if len(h1) != 8 {
		t.Errorf("expected hash length 8, got %d", len(h1))
	}
}

func TestShortHash_Different(t *testing.T) {
	h1 := shortHash("input A")
	h2 := shortHash("input B")
	if h1 == h2 {
		t.Errorf("different inputs should produce different hashes: %q == %q", h1, h2)
	}
}

func TestBuildPromptCacheKey_UsesOsGetwd(t *testing.T) {
	// BuildPromptCacheKey (public) should not panic
	key := BuildPromptCacheKey("gpt-4o", "test prompt")
	if !strings.HasPrefix(key, "xelyon:") {
		t.Errorf("expected key to start with 'xelyon:', got %q", key)
	}
	parts := strings.Split(key, ":")
	if len(parts) != 4 {
		t.Errorf("expected 4 parts, got %d: %q", len(parts), key)
	}
}

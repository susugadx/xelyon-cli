package openai

import (
	"strings"
	"testing"
)

func TestBuildPromptCacheKey_Format(t *testing.T) {
	key := buildPromptCacheKeyWithCwd("/home/user/project", "gpt-4o", "You are a helpful assistant")

	parts := strings.Split(key, ":")
	if len(parts) != 6 {
		t.Fatalf("expected 6 parts separated by ':', got %d: %q", len(parts), key)
	}
	if parts[0] != "xelyon" {
		t.Errorf("expected prefix 'xelyon', got %q", parts[0])
	}
	if parts[1] != "v2" {
		t.Errorf("expected version 'v2', got %q", parts[1])
	}
	if len(parts[2]) != 8 {
		t.Errorf("expected cwd_hash length 8, got %d: %q", len(parts[2]), parts[2])
	}
	if parts[3] != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", parts[3])
	}
	if len(parts[4]) != 8 || len(parts[5]) != 8 {
		t.Errorf("expected hash lengths 8, got %q / %q", parts[4], parts[5])
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

func TestBuildPromptCacheKey_IgnoresProjectMapSection(t *testing.T) {
	promptA := "base\n\n## Project Map\nTop-level files:\n- main.go"
	promptB := "base\n\n## Project Map\nTop-level files:\n- other.go"

	key1 := buildPromptCacheKeyWithCwd("/p", "m", promptA)
	key2 := buildPromptCacheKeyWithCwd("/p", "m", promptB)

	if key1 != key2 {
		t.Errorf("project map differences should not change key: %q != %q", key1, key2)
	}
}

func TestBuildPromptCacheKey_ProjectConfigChangesKey(t *testing.T) {
	promptA := "base\n<!-- PROJECT_CONFIG_START -->rule A<!-- PROJECT_CONFIG_END -->"
	promptB := "base\n<!-- PROJECT_CONFIG_START -->rule B<!-- PROJECT_CONFIG_END -->"

	key1 := buildPromptCacheKeyWithCwd("/p", "m", promptA)
	key2 := buildPromptCacheKeyWithCwd("/p", "m", promptB)

	if key1 == key2 {
		t.Errorf("project config differences should change key: %q == %q", key1, key2)
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
	if len(parts) != 6 {
		t.Errorf("expected 6 parts, got %d: %q", len(parts), key)
	}
}

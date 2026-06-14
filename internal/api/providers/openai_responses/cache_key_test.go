package openairesponses

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestBuildPromptCacheKeyWithCwdDelegatesOpenAICompatRules(t *testing.T) {
	promptA := "base prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- main.go")
	promptB := "base prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- other.go")

	keyA := BuildPromptCacheKeyWithCwd("/repo", "gpt-5.4", promptA)
	keyB := BuildPromptCacheKeyWithCwd("/repo", "gpt-5.4", promptB)
	if keyA != keyB {
		t.Fatalf("project map changed Responses cache key: %q != %q", keyA, keyB)
	}

	parts := strings.Split(keyA, ":")
	if len(parts) != 6 || parts[0] != "xelyon" || parts[1] != "v2" || parts[3] != "gpt-5.4" {
		t.Fatalf("cache key = %q, want xelyon:v2 format with model part", keyA)
	}
}

func TestBuildPromptCacheKeyUsesCurrentWorkingDirectory(t *testing.T) {
	key := BuildPromptCacheKey("gpt-5.4", "system prompt")
	if !strings.HasPrefix(key, "xelyon:v2:") {
		t.Fatalf("BuildPromptCacheKey() = %q, want xelyon:v2 prefix", key)
	}
}

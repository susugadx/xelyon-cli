package openaicompat

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestBuildPromptCacheKeyWithCwdFormat(t *testing.T) {
	key := BuildPromptCacheKeyWithCwd("/home/user/project", "provider/model", "system prompt")

	parts := strings.Split(key, ":")
	if len(parts) != 6 {
		t.Fatalf("key parts = %d, want 6: %q", len(parts), key)
	}
	if parts[0] != "xelyon" || parts[1] != "v2" {
		t.Fatalf("key prefix = %q:%q, want xelyon:v2", parts[0], parts[1])
	}
	if parts[3] != "provider/model" {
		t.Fatalf("model part = %q, want provider/model", parts[3])
	}
	if len(parts[2]) != 8 || len(parts[4]) != 8 || len(parts[5]) != 8 {
		t.Fatalf("hash parts = %q/%q/%q, want 8 hex chars", parts[2], parts[4], parts[5])
	}
}

func TestBuildPromptCacheKeyWithCwdIgnoresProjectMapSection(t *testing.T) {
	promptA := "base prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- main.go")
	promptB := "base prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- other.go")

	keyA := BuildPromptCacheKeyWithCwd("/repo", "model", promptA)
	keyB := BuildPromptCacheKeyWithCwd("/repo", "model", promptB)

	if keyA != keyB {
		t.Fatalf("project map changed cache key: %q != %q", keyA, keyB)
	}
}

func TestBuildPromptCacheKeyWithCwdProjectConfigChangesProjectHash(t *testing.T) {
	promptA := "base\n<!-- PROJECT_CONFIG_START -->rule A<!-- PROJECT_CONFIG_END -->"
	promptB := "base\n<!-- PROJECT_CONFIG_START -->rule B<!-- PROJECT_CONFIG_END -->"

	keyA := BuildPromptCacheKeyWithCwd("/repo", "model", promptA)
	keyB := BuildPromptCacheKeyWithCwd("/repo", "model", promptB)

	if keyA == keyB {
		t.Fatalf("project config should change cache key: %q", keyA)
	}
}

func TestBuildPromptCacheKeyWithCwdInputBoundariesAffectKey(t *testing.T) {
	base := BuildPromptCacheKeyWithCwd("/repo", "model", "system prompt")
	tests := []struct {
		name string
		key  string
	}{
		{name: "cwd", key: BuildPromptCacheKeyWithCwd("/other", "model", "system prompt")},
		{name: "model", key: BuildPromptCacheKeyWithCwd("/repo", "other-model", "system prompt")},
		{name: "core prompt", key: BuildPromptCacheKeyWithCwd("/repo", "model", "different system prompt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key == base {
				t.Fatalf("%s change did not affect cache key: %q", tt.name, base)
			}
		})
	}
}

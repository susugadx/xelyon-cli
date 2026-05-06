package kimi

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

func TestBuildKimiPromptCacheKey_UsesSessionScope(t *testing.T) {
	ctx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: " session-1 "})

	got := buildKimiPromptCacheKey(ctx, "kimi-k2.6", "System prompt")
	fallback := openaicompat.BuildPromptCacheKey("kimi-k2.6", "System prompt")

	if !strings.HasPrefix(got, "xelyon:kimi:v1:") {
		t.Fatalf("key = %q, want xelyon:kimi:v1 prefix", got)
	}
	assertKimiPromptCacheKeyHashLengths(t, got)
	if got == fallback {
		t.Fatalf("key = %q, want session-aware key distinct from fallback", got)
	}
}

func TestBuildKimiPromptCacheKey_SessionScopeIgnoresPromptAndProjectMap(t *testing.T) {
	ctx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "session-1"})
	promptA := "base prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- a.go")
	promptB := "changed prompt\n\n" + prompt.BuildProjectMapSection("## Project Map\n- b.go")

	keyA := buildKimiPromptCacheKey(ctx, "kimi-k2.6", promptA)
	keyB := buildKimiPromptCacheKey(ctx, "kimi-k2.6", promptB)

	if keyA != keyB {
		t.Fatalf("keys differ for prompt/project map changes: %q != %q", keyA, keyB)
	}
}

func TestBuildKimiPromptCacheKey_ModelAndTaskScopeAffectKey(t *testing.T) {
	sessionCtx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "session-1"})
	taskCtxA := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "session-1", TaskID: "task-a"})
	taskCtxB := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "session-1", TaskID: "task-b"})

	modelA := buildKimiPromptCacheKey(sessionCtx, "kimi-k2.6", "same prompt")
	modelB := buildKimiPromptCacheKey(sessionCtx, "kimi-k2.5", "same prompt")
	if modelA == modelB {
		t.Fatalf("different models produced same key: %q", modelA)
	}

	taskA := buildKimiPromptCacheKey(taskCtxA, "kimi-k2.6", "same prompt")
	taskB := buildKimiPromptCacheKey(taskCtxB, "kimi-k2.6", "same prompt")
	if taskA == taskB {
		t.Fatalf("different task IDs produced same key: %q", taskA)
	}
}

func TestBuildKimiPromptCacheKey_FallsBackWithoutSessionScope(t *testing.T) {
	got := buildKimiPromptCacheKey(context.Background(), "kimi-k2.6", "System prompt")
	want := openaicompat.BuildPromptCacheKey("kimi-k2.6", "System prompt")
	if got != want {
		t.Fatalf("key = %q, want fallback %q", got, want)
	}

	taskOnlyCtx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{TaskID: "task-only"})
	got = buildKimiPromptCacheKey(taskOnlyCtx, "kimi-k2.6", "System prompt")
	if got != want {
		t.Fatalf("task-only key = %q, want fallback %q", got, want)
	}
}

func assertKimiPromptCacheKeyHashLengths(t *testing.T, key string) {
	t.Helper()
	parts := strings.Split(key, ":")
	if len(parts) != 5 {
		t.Fatalf("key parts = %d, want 5 for %q", len(parts), key)
	}
	if len(parts[3]) != 24 || len(parts[4]) != 24 {
		t.Fatalf("hash lengths = %d/%d, want 24/24 for %q", len(parts[3]), len(parts[4]), key)
	}
}

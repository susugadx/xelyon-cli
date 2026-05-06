package api

import (
	"context"
	"testing"
)

func TestPromptCacheScopeContext_RoundTripAndTrim(t *testing.T) {
	ctx := WithPromptCacheScope(context.Background(), PromptCacheScope{
		SessionID: " session-1 ",
		TaskID:    " task-1 ",
	})

	got, ok := PromptCacheScopeFromContext(ctx)
	if !ok {
		t.Fatal("PromptCacheScopeFromContext() ok = false, want true")
	}
	if got.SessionID != "session-1" || got.TaskID != "task-1" {
		t.Fatalf("scope = %+v, want trimmed session/task", got)
	}
}

func TestPromptCacheScopeContext_SessionOnly(t *testing.T) {
	ctx := WithPromptCacheScope(context.Background(), PromptCacheScope{SessionID: " session-1 "})

	got, ok := PromptCacheScopeFromContext(ctx)
	if !ok {
		t.Fatal("PromptCacheScopeFromContext() ok = false, want true")
	}
	if got.SessionID != "session-1" || got.TaskID != "" {
		t.Fatalf("scope = %+v, want session only", got)
	}
}

func TestPromptCacheScopeContext_EmptyScopeClearsExistingScope(t *testing.T) {
	ctx := WithPromptCacheScope(context.Background(), PromptCacheScope{SessionID: "session-1", TaskID: "task-1"})
	ctx = WithPromptCacheScope(ctx, PromptCacheScope{})

	got, ok := PromptCacheScopeFromContext(ctx)
	if ok {
		t.Fatalf("PromptCacheScopeFromContext() = %+v, true; want empty false", got)
	}
}

func TestPromptCacheScopeContext_NilContext(t *testing.T) {
	ctx := WithPromptCacheScope(nil, PromptCacheScope{SessionID: "session-1"})

	got, ok := PromptCacheScopeFromContext(ctx)
	if !ok {
		t.Fatal("PromptCacheScopeFromContext() ok = false, want true")
	}
	if got.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", got.SessionID)
	}
}

package search

import (
	"context"
	"errors"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestWebSearchRequestContext_UsesEffectiveContextForCancellationAndPromptCacheScope(t *testing.T) {
	baseCtx := api.WithPromptCacheScope(context.Background(), api.PromptCacheScope{SessionID: "cancel-session"})
	baseCtx, cancel := context.WithCancel(baseCtx)
	cancel()

	cfg := config.DefaultConfig()
	var attributed api.Usage
	requestCtx := webSearchRequestContext(tools.ExecutionContext{
		Context: baseCtx,
		Config:  cfg,
		UsageAttribution: func(provider, model string, usage api.Usage) {
			if provider != "kimi" || model != "kimi-k2.6" {
				t.Fatalf("usage owner = %s/%s, want kimi/kimi-k2.6", provider, model)
			}
			attributed = usage
		},
	}, cfg, "kimi", "kimi-k2.6")

	if !errors.Is(requestCtx.Err(), context.Canceled) {
		t.Fatalf("request context Err() = %v, want context.Canceled", requestCtx.Err())
	}
	scope, ok := api.PromptCacheScopeFromContext(requestCtx)
	if !ok || scope.SessionID != "cancel-session" {
		t.Fatalf("PromptCacheScopeFromContext() = %+v, %t; want cancel-session", scope, ok)
	}
	callback := websearch.UsageCallbackFromContext(requestCtx)
	if callback == nil {
		t.Fatal("UsageCallbackFromContext() = nil, want callback")
	}
	callback(api.Usage{WebSearchCalls: 1})
	if attributed.WebSearchCalls != 1 {
		t.Fatalf("attributed usage = %+v, want one web search call", attributed)
	}
}

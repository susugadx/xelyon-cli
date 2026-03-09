package config

import (
	"context"
	"testing"
)

func TestWithContextAndFromContext_PreserveInjectedConfig(t *testing.T) {
	injected := DefaultConfig()
	injected.DefaultModel = "injected-model"

	ctx := WithContext(context.Background(), injected)
	got := FromContext(ctx)
	if got != injected {
		t.Fatalf("FromContext() should preserve injected config")
	}
}

func TestResolveContext_UsesFallbackWhenContextMissing(t *testing.T) {
	fallback := DefaultConfig()
	fallback.DefaultModel = "fallback-model"

	got := ResolveContext(context.Background(), fallback)
	if got != fallback {
		t.Fatalf("ResolveContext() should use fallback config")
	}
}

func TestResolveContext_UsesDefaultConfigWhenNil(t *testing.T) {
	got := ResolveContext(context.TODO(), nil)
	if got == nil {
		t.Fatal("ResolveContext(context.TODO(), nil) returned nil config")
	}
	if got.DefaultModel != DefaultConfig().DefaultModel {
		t.Fatalf("default model = %q, want %q", got.DefaultModel, DefaultConfig().DefaultModel)
	}
}

package config

import (
	"context"
	"testing"
)

func TestWithGlobalFallback_PreservesInjectedConfig(t *testing.T) {
	originalGlobal := globalConfig
	t.Cleanup(func() {
		globalConfig = originalGlobal
	})

	globalConfig = DefaultConfig()
	globalConfig.DefaultModel = "global-model"

	injected := DefaultConfig()
	injected.DefaultModel = "injected-model"

	ctx := WithContext(context.Background(), injected)
	got := FromContext(WithGlobalFallback(ctx))
	if got != injected {
		t.Fatalf("WithGlobalFallback should preserve injected config")
	}
}

func TestWithGlobalFallback_UsesGlobalConfig(t *testing.T) {
	originalGlobal := globalConfig
	t.Cleanup(func() {
		globalConfig = originalGlobal
	})

	global := DefaultConfig()
	global.DefaultModel = "global-model"
	globalConfig = global

	got := FromContext(WithGlobalFallback(context.Background()))
	if got != global {
		t.Fatalf("WithGlobalFallback should use current global config")
	}
}

func TestWithGlobalFallback_UsesDefaultConfigWithoutInitializingGlobal(t *testing.T) {
	originalGlobal := globalConfig
	t.Cleanup(func() {
		globalConfig = originalGlobal
	})

	globalConfig = nil

	got := FromContext(WithGlobalFallback(context.TODO()))
	if got == nil {
		t.Fatal("WithGlobalFallback(context.TODO()) returned nil config")
	}
	if got.DefaultModel != DefaultConfig().DefaultModel {
		t.Fatalf("default model = %q, want %q", got.DefaultModel, DefaultConfig().DefaultModel)
	}
	if globalConfig != nil {
		t.Fatal("WithGlobalFallback should not initialize globalConfig when it is unset")
	}
}

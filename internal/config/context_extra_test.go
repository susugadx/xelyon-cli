package config

import (
	"context"
	"testing"
)

func TestWithContextLookupAndFromContext_NilCases(t *testing.T) {
	base := context.Background()
	if got := WithContext(base, nil); got != base {
		t.Fatal("WithContext(ctx, nil) should return the original context")
	}

	if cfg, ok := LookupContext(base); ok || cfg != nil {
		t.Fatalf("LookupContext(context.Background()) = (%v, %v), want (nil, false)", cfg, ok)
	}

	defaultCfg := FromContext(nil)
	if defaultCfg == nil {
		t.Fatal("FromContext(nil) returned nil")
	}
	if defaultCfg.DefaultModel != DefaultConfig().DefaultModel {
		t.Fatalf("FromContext(nil).DefaultModel = %q, want %q", defaultCfg.DefaultModel, DefaultConfig().DefaultModel)
	}
}

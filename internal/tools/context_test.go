package tools

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type testContextKey struct{}

func TestExecutionContextZeroValue_UsesIsolatedDefaults(t *testing.T) {
	ctx := ExecutionContext{}

	if ctx.EffectiveToolCache() != nil {
		t.Fatal("expected default execution context to omit tool cache")
	}
	if ctx.ConfirmOptions().AutoApprove {
		t.Fatal("expected default execution context to disable auto-approve")
	}
	if ctx.EffectiveConfig().DefaultProvider != config.DefaultConfig().DefaultProvider {
		t.Fatalf("expected isolated default config, got %q", ctx.EffectiveConfig().DefaultProvider)
	}
	if logger := ctx.EffectiveAuditLogger(); logger == nil {
		t.Fatal("expected audit logger to be non-nil")
	}
	if ctx.EffectiveContext() == nil {
		t.Fatal("expected execution context to expose a non-nil context")
	}
}

func TestExecutionContext_EffectiveContextPreservesInjectedContext(t *testing.T) {
	base := context.WithValue(context.Background(), testContextKey{}, "value")
	ctx := ExecutionContext{Context: base}

	if got := ctx.EffectiveContext(); got != base {
		t.Fatal("expected injected context to be preserved")
	}
}

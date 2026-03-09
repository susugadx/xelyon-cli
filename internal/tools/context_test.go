package tools

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDefaultExecutionContext_UsesIsolatedDefaults(t *testing.T) {
	ctx := DefaultExecutionContext()

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
}

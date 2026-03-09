package tools

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/audit"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDefaultExecutionContext_UsesIsolatedDefaults(t *testing.T) {
	originalConfig := config.GetGlobalConfig()
	customConfig := config.CloneConfig(originalConfig)
	customConfig.DefaultProvider = "gemini"
	config.SetGlobalConfig(customConfig)
	t.Cleanup(func() {
		config.SetGlobalConfig(originalConfig)
	})

	globalLogger := audit.NewLoggerWithPath(filepath.Join(t.TempDir(), "audit.jsonl"), true)
	audit.SetGlobalLogger(globalLogger)
	t.Cleanup(func() {
		audit.SetGlobalLogger(nil)
	})

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
	} else if logger == globalLogger {
		t.Fatal("expected default execution context to avoid global audit logger")
	}
}

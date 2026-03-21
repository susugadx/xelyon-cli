package tools

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func TestExecutionContext_EffectiveProviderPreservesInjectedProvider(t *testing.T) {
	provider := &contextTestProvider{}
	ctx := ExecutionContext{Provider: provider}

	if got := ctx.EffectiveProvider(); got != provider {
		t.Fatal("expected injected provider to be preserved")
	}
}

type contextTestProvider struct{}

func (p *contextTestProvider) Name() string { return "context-test" }

func (p *contextTestProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "", nil
}

func (p *contextTestProvider) SupportsImages() bool { return false }

func (p *contextTestProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "", nil
}

func (p *contextTestProvider) IsFunctionCallingEnabled() bool { return true }

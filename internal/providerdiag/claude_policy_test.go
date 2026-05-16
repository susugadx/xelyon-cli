package providerdiag

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestClaudeCatalogPolicyUsesClaudeMetadataOnly(t *testing.T) {
	cfg := ProviderDiagnosticPolicyConfig(config.DefaultConfig(), ProviderDiagnosticPolicyConfigOptions{
		Provider:     "claude",
		Model:        "corp-claude",
		CatalogModel: "claude-sonnet-4-6",
	})

	policy := ClaudeCatalogPolicy(cfg, "corp-claude", "claude-sonnet-4-6")
	if !policy.ContextWindowKnown || policy.ContextWindowTokens == 0 {
		t.Fatalf("ContextWindow = %d known=%t, want Claude metadata", policy.ContextWindowTokens, policy.ContextWindowKnown)
	}
	if !policy.MaxOutput.Available || policy.MaxOutput.Tokens == 0 {
		t.Fatalf("MaxOutput = %+v, want Claude max output metadata", policy.MaxOutput)
	}
	if policy.Pricing.PricingUnavailable {
		t.Fatalf("PricingUnavailable = true, want Claude pricing metadata")
	}

	for _, catalogModel := range []string{"claude-sonnet-4-20250514", "claude-3-opus-20240229"} {
		t.Run(catalogModel, func(t *testing.T) {
			if !IsProviderCatalogModelKnown("claude", catalogModel) {
				t.Fatalf("IsProviderCatalogModelKnown(claude, %q) = false, want true", catalogModel)
			}
			cfg := ProviderDiagnosticPolicyConfig(config.DefaultConfig(), ProviderDiagnosticPolicyConfigOptions{
				Provider:     "claude",
				Model:        "corp-claude",
				CatalogModel: catalogModel,
			})
			policy := ClaudeCatalogPolicy(cfg, "corp-claude", catalogModel)
			if !policy.ContextWindowKnown || policy.ContextWindowTokens == 0 {
				t.Fatalf("ContextWindow = %d known=%t, want Claude metadata", policy.ContextWindowTokens, policy.ContextWindowKnown)
			}
			if policy.Pricing.PricingUnavailable {
				t.Fatalf("PricingUnavailable = true, want Claude pricing metadata")
			}
		})
	}

	other := ClaudeCatalogPolicy(cfg, "corp-claude", "gpt-5.5")
	if other.ContextWindowKnown || other.ContextWindowTokens != 0 {
		t.Fatalf("non-Claude context window = %d known=%t, want unknown", other.ContextWindowTokens, other.ContextWindowKnown)
	}
	if other.MaxOutput.Available || other.MaxOutput.Tokens != 0 {
		t.Fatalf("non-Claude max output = %+v, want unavailable", other.MaxOutput)
	}
	if !other.Pricing.PricingUnavailable {
		t.Fatalf("non-Claude pricing should be unavailable")
	}
	if detail := other.ClaudeDetail(); !strings.Contains(detail, "context_window=unknown") || !strings.Contains(detail, "max_output_tokens=unknown") {
		t.Fatalf("ClaudeDetail() = %q, want unknown metadata", detail)
	}
}

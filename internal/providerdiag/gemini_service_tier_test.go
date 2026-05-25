package providerdiag

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestNewGeminiServiceTierSnapshot(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority
	usage := &api.Usage{BillingServiceTier: config.GeminiServiceTierStandard}

	got := NewGeminiServiceTierSnapshot(cfg, usage)
	if got.ConfiguredTier != config.GeminiServiceTierPriority ||
		got.RequestBodyTier != config.GeminiServiceTierPriority ||
		got.PricingFamily != "gemini_priority" ||
		got.BillingTier != config.GeminiServiceTierStandard ||
		got.BillingPricingFamily != "gemini" {
		t.Fatalf("NewGeminiServiceTierSnapshot() = %+v, want priority request with standard billing", got)
	}
	for _, want := range []string{
		"configured=priority",
		"request_body=priority",
		"pricing_family=gemini_priority",
		"billing=standard",
		"billing_pricing_family=gemini",
	} {
		if !strings.Contains(got.Detail(), want) {
			t.Fatalf("Detail() = %q, want %q", got.Detail(), want)
		}
	}
}

func TestNewGeminiServiceTierSnapshotStandardOmitsRequestBodyTier(t *testing.T) {
	got := NewGeminiServiceTierSnapshot(config.DefaultConfig(), nil)
	if got.ConfiguredTier != config.GeminiServiceTierStandard ||
		got.RequestBodyTier != geminiRequestBodyTierOmitted ||
		got.PricingFamily != "gemini" ||
		got.BillingTier != "" {
		t.Fatalf("NewGeminiServiceTierSnapshot(default) = %+v, want standard omitted request tier", got)
	}
}

func TestNewGeminiServiceTierSnapshotMixedBillingTier(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierFlex

	got := NewGeminiServiceTierSnapshot(cfg, &api.Usage{BillingServiceTier: "mixed"})
	if got.BillingTier != "mixed" || got.BillingPricingFamily != "mixed" {
		t.Fatalf("NewGeminiServiceTierSnapshot(mixed) = %+v, want mixed billing tier", got)
	}
}

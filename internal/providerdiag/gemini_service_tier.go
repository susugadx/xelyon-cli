package providerdiag

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
)

const (
	geminiRequestBodyTierOmitted = "omitted"
	geminiBillingTierMixed       = "mixed"
)

// GeminiServiceTierSnapshot は Gemini BYOK service tier の request / pricing / billing 診断用 snapshot。
type GeminiServiceTierSnapshot struct {
	ConfiguredTier       string `json:"configured_tier"`
	RequestBodyTier      string `json:"request_body_tier"`
	PricingFamily        string `json:"pricing_family"`
	BillingTier          string `json:"billing_tier,omitempty"`
	BillingPricingFamily string `json:"billing_pricing_family,omitempty"`
}

// NewGeminiServiceTierSnapshot は config と観測 usage から service tier 診断値を構築する。
func NewGeminiServiceTierSnapshot(cfg *config.Config, usage *api.Usage) GeminiServiceTierSnapshot {
	configured := cfg.GeminiServiceTier()
	requestBodyTier := config.GeminiRequestServiceTier(cfg)
	if requestBodyTier == "" {
		requestBodyTier = geminiRequestBodyTierOmitted
	}

	snapshot := GeminiServiceTierSnapshot{
		ConfiguredTier:  configured,
		RequestBodyTier: requestBodyTier,
		PricingFamily:   cost.GeminiPricingFamilyForServiceTier(configured),
	}
	if usage == nil {
		return snapshot
	}

	billing := strings.TrimSpace(usage.BillingServiceTier)
	switch {
	case billing == "":
		return snapshot
	case strings.EqualFold(billing, geminiBillingTierMixed):
		snapshot.BillingTier = geminiBillingTierMixed
		snapshot.BillingPricingFamily = geminiBillingTierMixed
	case config.IsValidGeminiServiceTier(billing):
		billing = config.NormalizeGeminiServiceTier(billing)
		snapshot.BillingTier = billing
		snapshot.BillingPricingFamily = cost.GeminiPricingFamilyForServiceTier(billing)
	}
	return snapshot
}

// Detail は doctor/status 表示用の安定した summary を返す。
func (s GeminiServiceTierSnapshot) Detail() string {
	detail := fmt.Sprintf(
		"configured=%s, request_body=%s, pricing_family=%s",
		s.ConfiguredTier,
		s.RequestBodyTier,
		s.PricingFamily,
	)
	if strings.TrimSpace(s.BillingTier) != "" {
		detail += fmt.Sprintf(", billing=%s, billing_pricing_family=%s", s.BillingTier, s.BillingPricingFamily)
	}
	return detail
}

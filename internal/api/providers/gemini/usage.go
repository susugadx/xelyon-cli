package gemini

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func geminiUsageMetadataToAPIUsage(metadata *GeminiUsageMetadata, billingServiceTier ...string) (api.Usage, bool) {
	if metadata == nil {
		return api.Usage{}, false
	}
	usage := api.Usage{
		InputTokens:       metadata.PromptTokenCount,
		OutputTokens:      metadata.CandidatesTokenCount,
		ThinkingTokens:    metadata.ThoughtsTokenCount,
		CachedInputTokens: metadata.CachedContentTokenCount,
	}
	if serviceTier := geminiBillingServiceTierFromUsageMetadata(metadata); serviceTier != "" {
		usage.BillingServiceTier = serviceTier
	} else if len(billingServiceTier) > 0 {
		usage.BillingServiceTier = billingServiceTier[0]
	}
	return usage, true
}

func (p *Provider) emitUsageMetadata(metadata *GeminiUsageMetadata, billingServiceTier ...string) {
	if p == nil || p.usageCallback == nil {
		return
	}
	usage, ok := geminiUsageMetadataToAPIUsage(metadata, billingServiceTier...)
	if !ok {
		return
	}
	p.usageCallback(usage)
}

func geminiBillingServiceTierFromUsageMetadata(metadata *GeminiUsageMetadata) string {
	if metadata == nil {
		return ""
	}
	return geminiBillingServiceTierFromValue(metadata.ServiceTier)
}

func geminiBillingServiceTierFromValue(value string) string {
	raw := strings.ToLower(strings.TrimSpace(value))
	if raw == "" {
		return ""
	}
	if raw == "unspecified" {
		return config.GeminiServiceTierStandard
	}
	serviceTier := config.NormalizeGeminiServiceTier(raw)
	if !config.IsValidGeminiServiceTier(serviceTier) {
		return ""
	}
	return serviceTier
}

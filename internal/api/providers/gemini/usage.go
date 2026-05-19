package gemini

import "github.com/susugadx/xelyon-cli/internal/api"

func geminiUsageMetadataToAPIUsage(metadata *GeminiUsageMetadata) (api.Usage, bool) {
	if metadata == nil {
		return api.Usage{}, false
	}
	return api.Usage{
		InputTokens:       metadata.PromptTokenCount,
		OutputTokens:      metadata.CandidatesTokenCount,
		ThinkingTokens:    metadata.ThoughtsTokenCount,
		CachedInputTokens: metadata.CachedContentTokenCount,
	}, true
}

func (p *Provider) emitUsageMetadata(metadata *GeminiUsageMetadata) {
	if p == nil || p.usageCallback == nil {
		return
	}
	usage, ok := geminiUsageMetadataToAPIUsage(metadata)
	if !ok {
		return
	}
	p.usageCallback(usage)
}

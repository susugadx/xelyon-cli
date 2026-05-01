package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type autoCompressionAction int

const (
	autoCompressionActionNone autoCompressionAction = iota
	autoCompressionActionRun
	autoCompressionActionWarnUnknownContext
)

type autoCompressionReason int

const (
	autoCompressionReasonNone autoCompressionReason = iota
	autoCompressionReasonPricingCliff
	autoCompressionReasonProviderThreshold
	autoCompressionReasonTokenThreshold
	autoCompressionReasonThresholdTokens
	autoCompressionReasonLocalThreshold
)

type autoCompressionDecision struct {
	action          autoCompressionAction
	reason          autoCompressionReason
	providerKey     string
	model           string
	currentTokens   int
	projectedTokens int
	thresholdTokens int
	costAware       bool
}

func (a *Agent) planAutoCompression(cfg *config.Config, providerKey string, currentTokens int) autoCompressionDecision {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	model := a.CurrentModel
	base := autoCompressionDecision{
		providerKey:   providerKey,
		model:         model,
		currentTokens: currentTokens,
	}

	if !cfg.Compression.Enabled {
		return base
	}

	if a.shouldSkipLocalAutoCompressionForServerCompaction(cfg) {
		return base
	}

	projectedTokens, costAwareCompress := shouldForceCompressForPricingCliffForConfig(cfg, providerKey, model, currentTokens, a.Stats)
	if costAwareCompress {
		base.action = autoCompressionActionRun
		base.reason = autoCompressionReasonPricingCliff
		base.projectedTokens = projectedTokens
		base.costAware = true
		return base
	}

	providerThreshold := GetProviderCompressThresholdWithConfig(cfg, providerKey, model)
	if providerThreshold > 0 {
		if currentTokens < providerThreshold {
			return base
		}
		base.action = autoCompressionActionRun
		base.reason = autoCompressionReasonProviderThreshold
		base.thresholdTokens = providerThreshold
		return base
	}

	if a.shouldSkipLocalAutoCompressionForClaudeCompaction() {
		return base
	}

	if cfg.Compression.TokenThreshold > 0 && currentTokens >= cfg.Compression.TokenThreshold {
		base.action = autoCompressionActionRun
		base.reason = autoCompressionReasonTokenThreshold
		base.thresholdTokens = cfg.Compression.TokenThreshold
		return base
	}

	if cfg.Compression.ThresholdTokens > 0 {
		if currentTokens < cfg.Compression.ThresholdTokens {
			return base
		}
		base.action = autoCompressionActionRun
		base.reason = autoCompressionReasonThresholdTokens
		base.thresholdTokens = cfg.Compression.ThresholdTokens
		return base
	}

	localThreshold, ok := localAutoCompressionTokenThresholdForConfig(cfg, providerKey, model)
	if !ok {
		base.action = autoCompressionActionWarnUnknownContext
		return base
	}
	if currentTokens < localThreshold {
		return base
	}

	base.action = autoCompressionActionRun
	base.reason = autoCompressionReasonLocalThreshold
	base.thresholdTokens = localThreshold
	return base
}

func (a *Agent) shouldSkipLocalAutoCompressionForServerCompaction(cfg *config.Config) bool {
	if a == nil || cfg == nil || !cfg.ResponsesServerCompactionEnabled() {
		return false
	}
	ridProvider, ok := a.CurrentProvider.(ResponseIDCapable)
	if !ok || !ridProvider.HasCachedResponseID() {
		return false
	}
	return cfg.IsProviderResponsesAPIRequest(a.ProviderName, a.sessionProviderConfigKey(cfg), a.CurrentModel)
}

func (a *Agent) shouldSkipLocalAutoCompressionForClaudeCompaction() bool {
	if a == nil {
		return false
	}
	requestCtx := a.requestContext(context.Background())
	if compactionProvider, ok := a.CurrentProvider.(api.ClaudeCompactionRuntimeCapable); ok {
		return compactionProvider.SupportsClaudeCompactionWithContext(requestCtx, a.CurrentModel)
	}
	if compactionProvider, ok := a.CurrentProvider.(api.ClaudeCompactionCapable); ok {
		return compactionProvider.SupportsClaudeCompaction()
	}
	return false
}

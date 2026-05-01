package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ResponseIDCapable は Responses API のキャッシュ機能を持つプロバイダー
type ResponseIDCapable interface {
	HasCachedResponseID() bool
	SetResponseID(id string)
	GetResponseID() string
}

func (a *Agent) runAutoCompression(costAwareCompress bool) bool {
	cfg := a.cfg()

	// Compact API を優先的に使用するか確認
	if cfg.Compression.PreferCompactAPI {
		if compactProvider, ok := a.CurrentProvider.(api.CompactCapable); ok {
			if compactProvider.SupportsCompact() {
				ctx := context.Background()
				if err := a.CompressWithCompactAPI(ctx); err == nil {
					metrics := OptimizationMetrics{CompactionCount: 1}
					if costAwareCompress {
						metrics.CostAwareCompressions = 1
					}
					a.addOptimizationMetrics(metrics)
					_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.enabled false")
					_, _ = fmt.Fprintln(a.output())
					return true
				}
				// Compact API 失敗時はLLMサマリーにフォールバック
				yellow.Fprintf(a.output(), "   ⚠️ Compact API failed, falling back to LLM summary...\n")
			}
		}
	}

	keepRecent := cfg.Compression.KeepRecent
	if keepRecent == 0 {
		keepRecent = 10
	}

	// 履歴が短すぎる場合はスキップ
	if len(a.History) <= keepRecent {
		_, _ = fmt.Fprintln(a.output(), "   Skipped: history too short")
		return false
	}

	beforeTokens := a.EstimateTokens()
	if err := a.CompressHistory(keepRecent); err != nil {
		yellow.Fprintf(a.output(), "   ⚠️ Auto-compress failed: %v\n", err)
		return false
	}
	afterTokens := a.EstimateTokens()

	// 結果を表示
	_, _ = fmt.Fprintf(a.output(), "   Before: %s tokens → After: %s tokens\n",
		formatNumber(beforeTokens), formatNumber(afterTokens))
	_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.enabled false")
	_, _ = fmt.Fprintln(a.output())

	metrics := OptimizationMetrics{CompactionCount: 1}
	if costAwareCompress {
		metrics.CostAwareCompressions = 1
	}
	a.addOptimizationMetrics(metrics)
	return true
}

// maybeAutoCompress は閾値を超えた場合に自動圧縮を実行
// 圧縮した場合は true を返す
func (a *Agent) maybeAutoCompress() bool {
	cfg := a.cfg()
	if !cfg.Compression.Enabled {
		return false
	}

	currentTokens := a.EstimateTokens()
	providerKey := a.sessionProviderConfigKey(cfg)
	decision := a.planAutoCompression(cfg, providerKey, currentTokens)
	return a.applyAutoCompressionDecision(decision)
}

func (a *Agent) applyAutoCompressionDecision(decision autoCompressionDecision) bool {
	switch decision.action {
	case autoCompressionActionRun:
		a.printAutoCompressionStart(decision)
		return a.runAutoCompression(decision.costAware)
	case autoCompressionActionWarnUnknownContext:
		a.warnAutoCompressUnknownContext(decision.providerKey, decision.model)
		return false
	default:
		return false
	}
}

func (a *Agent) printAutoCompressionStart(decision autoCompressionDecision) {
	switch decision.reason {
	case autoCompressionReasonPricingCliff:
		cyan.Fprintf(a.output(),
			"\n🗜️ Auto-compressing around pricing cliff (%dK → %dK projected)...\n",
			decision.currentTokens/1000,
			decision.projectedTokens/1000,
		)
	case autoCompressionReasonProviderThreshold:
		cyan.Fprintf(a.output(),
			"\n🗜️ Auto-compressing at custom provider threshold (%dK >= %dK threshold)...\n",
			decision.currentTokens/1000,
			decision.thresholdTokens/1000,
		)
	case autoCompressionReasonTokenThreshold:
		cyan.Fprintf(a.output(),
			"\n🗜️ Auto-compressing: context %dK exceeds %dK custom threshold...\n",
			decision.currentTokens/1000,
			decision.thresholdTokens/1000,
		)
	case autoCompressionReasonThresholdTokens, autoCompressionReasonLocalThreshold:
		percentage := a.GetTokenUsagePercentage()
		cyan.Fprintf(a.output(),
			"\n🗜️ Auto-compressing history (%dK >= %dK threshold, %.0f%% context used)...\n",
			decision.currentTokens/1000,
			decision.thresholdTokens/1000,
			percentage,
		)
	}
}

func (a *Agent) warnAutoCompressUnknownContext(provider, model string) {
	if a == nil {
		return
	}
	key := provider + "\x00" + model
	if a.autoCompressUnknownContextWarningKey == key {
		return
	}
	a.autoCompressUnknownContextWarningKey = key
	yellow.Fprintf(a.output(),
		"⚠️ Auto-compress skipped: context window is unknown for %s/%s. Set provider_models catalog_model or compression.provider_thresholds to enable local auto-compress.\n",
		provider,
		model,
	)
}

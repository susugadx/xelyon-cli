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

// ResponsesServerCompactionLocalSkipCapable は
// 直近 request の server-side compaction 判定に基づく local auto-compress 抑止可否を返す。
type ResponsesServerCompactionLocalSkipCapable interface {
	ShouldSkipLocalAutoCompressionForServerCompaction() bool
}

func (a *Agent) runAutoCompression(decision autoCompressionDecision) bool {
	cfg := a.cfg()
	keepRecent := normalizedAutoCompressionKeepRecent(cfg.Compression.KeepRecent)
	beforeTokens := decision.currentTokens
	if beforeTokens == 0 {
		beforeTokens = a.EstimateTokens()
	}
	display := compressionDisplayOperation{}

	// Compact API を優先的に使用するか確認
	if cfg.Compression.PreferCompactAPI {
		if compactProvider, ok := a.CurrentProvider.(api.CompactCapable); ok {
			if compactProvider.SupportsCompact() {
				display = a.beginCompressionDisplay(
					compressionDisplayModeCompactAPI,
					compressionDisplayReasonAuto,
					keepRecent,
					beforeTokens,
				)
				ctx := context.Background()
				result, err := a.compressWithCompactAPI(ctx, compressCompactOptions{
					displayReason:      compressionDisplayReasonAuto,
					suppressTUIDisplay: true,
				})
				if err == nil {
					a.finishCompressionDisplay(display, compactResultOutputTokens(result), nil)
					metrics := OptimizationMetrics{CompactionCount: 1}
					if decision.costAware {
						metrics.CostAwareCompressions = 1
					}
					a.addOptimizationMetrics(metrics)
					if a.shouldPrintCompressionOutput() {
						_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.enabled false")
						_, _ = fmt.Fprintln(a.output())
					}
					return true
				}
				// Compact API 失敗時はLLMサマリーにフォールバック
				if a.usesTUICompressionDisplay() {
					display = a.updateCompressionDisplay(display, compressionDisplayModeHistory, "Compact API failed; falling back to history summary.")
				} else {
					yellow.Fprintf(a.output(), "   ⚠️ Compact API failed, falling back to LLM summary...\n")
				}
			}
		}
	}

	if !display.active {
		display = a.beginCompressionDisplay(
			compressionDisplayModeHistory,
			compressionDisplayReasonAuto,
			keepRecent,
			beforeTokens,
		)
	}

	// 履歴が短すぎる場合はスキップ
	if len(a.History) <= keepRecent {
		if a.shouldPrintCompressionOutput() {
			_, _ = fmt.Fprintln(a.output(), "   Skipped: history too short")
		}
		a.finishCompressionDisplaySkipped(display, "history too short")
		return false
	}

	result := a.runAutoCompressionHistorySummary(decision, display, keepRecent, beforeTokens, func(opts compressHistoryOptions) error {
		return a.compressHistory(keepRecent, opts)
	})
	return result.compressed
}

func normalizedAutoCompressionKeepRecent(keepRecent int) int {
	if keepRecent == 0 {
		return 10
	}
	return keepRecent
}

type inTurnAutoCompressionPlan struct {
	ctx         context.Context
	decision    autoCompressionDecision
	historyPlan compressionHistoryPlan
	keepRecent  int
}

type inTurnAutoCompressionOptions struct {
	persistHistory []api.Message
}

type inTurnAutoCompressionResult struct {
	attempted                       bool
	compressed                      bool
	compressedCurrentTurnStartIndex int
	requestErr                      error
}

func (a *Agent) maybeAutoCompressDuringTurn(ctx context.Context, currentTurnStartIndex int, state *autoCompressionTurnState) inTurnAutoCompressionResult {
	return a.maybeAutoCompressDuringTurnWithOptions(ctx, currentTurnStartIndex, state, inTurnAutoCompressionOptions{})
}

func (a *Agent) maybeAutoCompressDuringTurnWithOptions(ctx context.Context, currentTurnStartIndex int, state *autoCompressionTurnState, opts inTurnAutoCompressionOptions) inTurnAutoCompressionResult {
	plan, ok := a.planInTurnAutoCompression(ctx, currentTurnStartIndex, state, opts)
	if !ok {
		return inTurnAutoCompressionResult{}
	}
	return a.runInTurnAutoCompression(plan, state)
}

func (a *Agent) planInTurnAutoCompression(ctx context.Context, currentTurnStartIndex int, state *autoCompressionTurnState, opts inTurnAutoCompressionOptions) (inTurnAutoCompressionPlan, bool) {
	if state == nil || state.attemptedThisTurn() {
		return inTurnAutoCompressionPlan{}, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return inTurnAutoCompressionPlan{}, false
	}

	cfg := a.cfg()
	if !cfg.Compression.Enabled {
		return inTurnAutoCompressionPlan{}, false
	}

	currentTokens := a.EstimateTokens()
	providerKey := a.sessionProviderConfigKey(cfg)
	decision := a.planAutoCompression(cfg, providerKey, currentTokens)
	if decision.action != autoCompressionActionRun {
		return inTurnAutoCompressionPlan{}, false
	}

	keepRecent := normalizedAutoCompressionKeepRecent(cfg.Compression.KeepRecent)
	historyPlan := a.compressionHistoryPlanForInTurn(currentTurnStartIndex, keepRecent)
	if len(opts.persistHistory) > 0 {
		historyPlan = a.compressionHistoryPlanForInTurnWithPersistHistory(currentTurnStartIndex, keepRecent, opts.persistHistory)
	}
	if !historyPlan.hasCompressibleHistory() {
		return inTurnAutoCompressionPlan{}, false
	}

	return inTurnAutoCompressionPlan{
		ctx:         ctx,
		decision:    decision,
		historyPlan: historyPlan,
		keepRecent:  keepRecent,
	}, true
}

func (a *Agent) runInTurnAutoCompression(plan inTurnAutoCompressionPlan, state *autoCompressionTurnState) inTurnAutoCompressionResult {
	beforeTokens := plan.decision.currentTokens
	if beforeTokens == 0 {
		beforeTokens = a.EstimateTokens()
	}

	a.printAutoCompressionStart(plan.decision)
	result := a.runAutoCompressionHistorySummary(plan.decision, compressionDisplayOperation{}, plan.keepRecent, beforeTokens, func(opts compressHistoryOptions) error {
		opts.baseContext = plan.ctx
		opts.onSummaryStart = func() {
			state.recordAttempt(false)
		}
		err := a.compressHistoryWithPlan(plan.historyPlan, opts)
		if err == nil {
			state.recordAttempt(true)
		}
		return err
	})

	return inTurnAutoCompressionResult{
		attempted:                       state.attemptedThisTurn(),
		compressed:                      result.compressed,
		compressedCurrentTurnStartIndex: compressedCurrentTurnStartIndexForResult(result, plan.historyPlan),
		requestErr:                      requestContextErr(plan.ctx),
	}
}

func compressedCurrentTurnStartIndexForResult(result autoCompressionRunResult, plan compressionHistoryPlan) int {
	if !result.compressed {
		return 0
	}
	return plan.compressedCurrentTurnStartIndex
}

type autoCompressionRunResult struct {
	compressed bool
}

func (a *Agent) runAutoCompressionHistorySummary(
	decision autoCompressionDecision,
	display compressionDisplayOperation,
	displayKeepRecent int,
	beforeTokens int,
	compress func(compressHistoryOptions) error,
) autoCompressionRunResult {
	if !display.active {
		display = a.beginCompressionDisplay(
			compressionDisplayModeHistory,
			compressionDisplayReasonAuto,
			displayKeepRecent,
			beforeTokens,
		)
	}

	if err := compress(compressHistoryOptions{
		displayReason:      compressionDisplayReasonAuto,
		suppressTUIDisplay: true,
	}); err != nil {
		if a.shouldPrintCompressionOutput() {
			yellow.Fprintf(a.output(), "   ⚠️ Auto-compress failed: %v\n", err)
		}
		a.finishCompressionDisplay(display, 0, err)
		return autoCompressionRunResult{}
	}
	afterTokens := a.EstimateTokens()
	a.finishCompressionDisplay(display, afterTokens, nil)

	if a.shouldPrintCompressionOutput() {
		_, _ = fmt.Fprintf(a.output(), "   Before: %s tokens → After: %s tokens\n",
			formatNumber(beforeTokens), formatNumber(afterTokens))
		_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.enabled false")
		_, _ = fmt.Fprintln(a.output())
	}

	metrics := OptimizationMetrics{CompactionCount: 1}
	if decision.costAware {
		metrics.CostAwareCompressions = 1
	}
	a.addOptimizationMetrics(metrics)
	return autoCompressionRunResult{compressed: true}
}

type autoCompressionAttemptResult struct {
	attempted  bool
	compressed bool
}

func (a *Agent) maybeAutoCompressAttempt() autoCompressionAttemptResult {
	cfg := a.cfg()
	if !cfg.Compression.Enabled {
		return autoCompressionAttemptResult{}
	}

	currentTokens := a.EstimateTokens()
	providerKey := a.sessionProviderConfigKey(cfg)
	decision := a.planAutoCompression(cfg, providerKey, currentTokens)
	return autoCompressionAttemptResult{
		attempted:  decision.action == autoCompressionActionRun,
		compressed: a.applyAutoCompressionDecision(decision),
	}
}

// maybeAutoCompress は閾値を超えた場合に自動圧縮を実行
// 圧縮した場合は true を返す
func (a *Agent) maybeAutoCompress() bool {
	return a.maybeAutoCompressAttempt().compressed
}

func (a *Agent) applyAutoCompressionDecision(decision autoCompressionDecision) bool {
	switch decision.action {
	case autoCompressionActionRun:
		a.printAutoCompressionStart(decision)
		return a.runAutoCompression(decision)
	case autoCompressionActionWarnUnknownContext:
		a.warnAutoCompressUnknownContext(decision.providerKey, decision.model)
		return false
	default:
		return false
	}
}

func (a *Agent) printAutoCompressionStart(decision autoCompressionDecision) {
	if !a.shouldPrintCompressionOutput() {
		return
	}
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

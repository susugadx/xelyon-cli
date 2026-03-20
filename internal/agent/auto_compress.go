package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
)

// ResponseIDCapable は Responses API のキャッシュ機能を持つプロバイダー
type ResponseIDCapable interface {
	HasCachedResponseID() bool
	SetResponseID(id string)
	GetResponseID() string
}

// defaultCompressionModel はプロバイダー別のデフォルト圧縮モデルを返す。
// compression.model が空の場合に使用する。
func defaultCompressionModel(providerName string) string {
	switch strings.ToLower(providerName) {
	case "openai":
		return "gpt-5-mini"
	case "gemini":
		return "gemini-3.1-flash-lite-preview"
	case "claude", "anthropic", "bedrock":
		return "claude-haiku-4-5"
	case "deepseek":
		return ""
	default:
		return ""
	}
}

// getCompressionModel は圧縮に使用するモデルを返す。
// 優先順位: config.compression.model > プロバイダー別デフォルト > 現在モデル。
func (a *Agent) getCompressionModel() string {
	cfg := a.cfg()
	if strings.EqualFold(cfg.Compression.Model, "main") {
		return a.CurrentModel
	}
	if cfg.Compression.Model != "" {
		return cfg.Compression.Model
	}
	if model := defaultCompressionModel(a.ProviderName); model != "" {
		return model
	}
	return a.CurrentModel
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
					_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.auto_compress false")
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
	_, _ = fmt.Fprintln(a.output(), "   💡 Disable with: xelyon config set compression.auto_compress false")
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
	if !cfg.Compression.AutoCompress {
		return false
	}

	// プロバイダ別コスト最適化閾値
	currentTokens := a.EstimateTokens()
	providerThreshold := GetProviderCompressThresholdWithConfig(cfg, a.ProviderName, a.CurrentModel)
	projectedTokens, costAwareCompress := shouldForceCompressForPricingCliff(a.ProviderName, a.CurrentModel, currentTokens, a.Stats)
	forceCompress := costAwareCompress
	if !forceCompress && providerThreshold > 0 && currentTokens >= providerThreshold {
		forceCompress = true
	}

	if !forceCompress {
		// Responses API で responseID がキャッシュされている場合は自動圧縮をスキップ
		// （サーバー側でコンテキスト管理、cached_tokens として課金）
		if ridProvider, ok := a.CurrentProvider.(ResponseIDCapable); ok {
			if ridProvider.HasCachedResponseID() {
				// キャッシュが効いているので圧縮不要
				return false
			}
		}

		// Claude Compaction が有効な場合は自動圧縮をスキップ
		requestCtx := a.requestContext(context.Background())
		if compactionProvider, ok := a.CurrentProvider.(api.ClaudeCompactionRuntimeCapable); ok {
			if compactionProvider.SupportsClaudeCompactionWithContext(requestCtx, a.CurrentModel) {
				return false
			}
		} else if compactionProvider, ok := a.CurrentProvider.(api.ClaudeCompactionCapable); ok {
			if compactionProvider.SupportsClaudeCompaction() {
				return false
			}
		}

		if cfg.Compression.TokenThreshold > 0 && currentTokens >= cfg.Compression.TokenThreshold {
			cyan.Fprintf(a.output(),
				"\n🗜️ Auto-compressing: context %dK exceeds %dK custom threshold...\n",
				currentTokens/1000,
				cfg.Compression.TokenThreshold/1000,
			)
			return a.runAutoCompression(false)
		}

		percentage := a.GetTokenUsagePercentage()

		// 閾値を取得（パーセントベース）
		threshold := float64(cfg.Compression.ThresholdPercent)
		if threshold == 0 {
			threshold = 80
		}

		// トークン数ベースの閾値が設定されている場合はそちらを優先
		if cfg.Compression.ThresholdTokens > 0 {
			if currentTokens < cfg.Compression.ThresholdTokens {
				return false
			}
		} else {
			// パーセントベース
			if percentage < threshold {
				return false
			}
		}
	}

	percentage := a.GetTokenUsagePercentage()

	// 圧縮実行（通知付き）
	if forceCompress {
		if costAwareCompress {
			cyan.Fprintf(a.output(),
				"\n🗜️ Auto-compressing before pricing cliff (%dK → %dK projected)...\n",
				currentTokens/1000,
				projectedTokens/1000,
			)
		} else {
			cyan.Fprintf(a.output(),
				"\n🗜️ Auto-compressing for cost optimization (%dK > %dK threshold)...\n",
				currentTokens/1000,
				providerThreshold/1000,
			)
		}
	} else {
		cyan.Fprintf(a.output(), "\n🗜️ Auto-compressing history (%.0f%% threshold reached)...\n", percentage)
	}

	return a.runAutoCompression(costAwareCompress)
}

// checkTokenWarning はトークン使用率をチェックして警告を表示
// 自動圧縮が有効な場合は表示しない（自動圧縮が処理するため）
func (a *Agent) checkTokenWarning() {
	cfg := a.cfg()

	// 自動圧縮が有効な場合は警告をスキップ（自動圧縮が処理する）
	if cfg.Compression.AutoCompress {
		return
	}

	percentage := a.GetTokenUsagePercentage()

	if percentage > 90 {
		yellow.Fprintln(a.output(), "⚠️ Token usage is at 90%. Consider using /compress")
	} else if percentage > 80 {
		_, _ = fmt.Fprintln(a.output(), "💡 Token usage is high (80%). /compress available if needed")
	}
}

// handleTokenLimitErrorWithWriter はトークン上限エラー時にユーザーへ案内を表示する。
// out は非nil前提（caller が a.output() を渡す）。
func handleTokenLimitErrorWithWriter(out io.Writer, err error) {
	if !token.IsTokenLimitError(err) {
		return
	}

	red.Fprintln(out, "❌ Token limit exceeded")
	yellow.Fprintln(out, "💡 Try: /compress to reduce history")
	yellow.Fprintln(out, "💡 Or:  /clear to start fresh")
}

// handleTokenLimitErrorWithRetry はトークン上限エラー時に自動圧縮してリトライ
// 戻り値: リトライ成功時はtrue、失敗時はfalse
func (a *Agent) handleTokenLimitErrorWithRetry(err error, retryFunc func() error, isPlanMode bool) bool {
	if !token.IsTokenLimitError(err) {
		return false
	}

	// リトライ制限（最大1回）
	if a.tokenLimitRetryCount >= 1 {
		// 2回目以降は通常のエラー表示
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// 通知表示
	cyan.Fprintln(a.output(), "\n⚡ トークン上限到達。自動圧縮して再実行します...")

	// LLMサマリー方式で圧縮（既存のCompressHistoryを使用）
	// keepRecentはデフォルト値10を使用
	keepRecent := 10
	cfg := a.cfg()
	if cfg.Compression.KeepRecent > 0 {
		keepRecent = cfg.Compression.KeepRecent
	}

	// 履歴が短すぎる場合は圧縮できない
	if len(a.History) <= keepRecent {
		yellow.Fprintln(a.output(), "⚠️  履歴が短すぎるため圧縮できません")
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// 圧縮実行
	yellow.Fprintln(a.output(), "🗜️  会話を圧縮中...")
	if err := a.CompressHistory(keepRecent); err != nil {
		red.Fprintf(a.output(), "❌ 自動圧縮に失敗しました: %v\n", err)
		handleTokenLimitErrorWithWriter(a.output(), err)
		return false
	}

	// リトライカウントをインクリメント
	a.tokenLimitRetryCount++

	// リトライ実行
	cyan.Fprintln(a.output(), "🔄 圧縮完了、再実行します...")
	if retryErr := retryFunc(); retryErr != nil {
		// リトライ後もエラーの場合は通常のエラー表示
		if token.IsTokenLimitError(retryErr) {
			red.Fprintln(a.output(), "❌ 圧縮後もトークン上限を超えています")
		}
		handleTokenLimitErrorWithWriter(a.output(), retryErr)
		return false
	}

	// リトライ成功
	green.Fprintln(a.output(), "✅ 自動圧縮＆リトライ成功")
	a.tokenLimitRetryCount = 0 // 成功したらリセット
	return true
}

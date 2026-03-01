package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// ResponseIDCapable は Responses API のキャッシュ機能を持つプロバイダー
type ResponseIDCapable interface {
	HasCachedResponseID() bool
}

// maybeAutoCompress は閾値を超えた場合に自動圧縮を実行
// 圧縮した場合は true を返す
func (a *Agent) maybeAutoCompress() bool {
	cfg := config.GetGlobalConfig()
	if !cfg.Compression.AutoCompress {
		return false
	}

	// プロバイダ別コスト最適化閾値
	providerThreshold := GetProviderCompressThreshold(a.ProviderName, a.CurrentModel)
	forceCompress := false
	if providerThreshold > 0 {
		currentTokens := a.EstimateTokens()
		if currentTokens >= providerThreshold {
			forceCompress = true
		}
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
		if compactionProvider, ok := a.CurrentProvider.(api.ClaudeCompactionCapable); ok {
			if compactionProvider.SupportsClaudeCompaction() {
				return false
			}
		}

		percentage := a.GetTokenUsagePercentage()

		// 閾値を取得（パーセントベース）
		threshold := float64(cfg.Compression.ThresholdPercent)
		if threshold == 0 {
			threshold = 80
		}

		// トークン数ベースの閾値が設定されている場合はそちらを優先
		if cfg.Compression.ThresholdTokens > 0 {
			currentTokens := a.EstimateTokens()
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
		cyan.Printf(
			"\n🗜️ Auto-compressing for cost optimization (%dK > %dK threshold)...\n",
			a.EstimateTokens()/1000,
			providerThreshold/1000,
		)
	} else {
		cyan.Printf("\n🗜️ Auto-compressing history (%.0f%% threshold reached)...\n", percentage)
	}

	// Compact API を優先的に使用するか確認
	if cfg.Compression.PreferCompactAPI {
		if compactProvider, ok := a.CurrentProvider.(api.CompactCapable); ok {
			if compactProvider.SupportsCompact() {
				ctx := context.Background()
				if err := a.CompressWithCompactAPI(ctx); err == nil {
					fmt.Println("   💡 Disable with: xelyon config set compression.auto_compress false")
					fmt.Println()
					return true
				}
				// Compact API 失敗時はLLMサマリーにフォールバック
				yellow.Printf("   ⚠️ Compact API failed, falling back to LLM summary...\n")
			}
		}
	}

	keepRecent := cfg.Compression.KeepRecent
	if keepRecent == 0 {
		keepRecent = 10
	}

	// 履歴が短すぎる場合はスキップ
	if len(a.History) <= keepRecent {
		fmt.Println("   Skipped: history too short")
		return false
	}

	beforeTokens := a.EstimateTokens()
	if err := a.CompressHistory(keepRecent); err != nil {
		yellow.Printf("   ⚠️ Auto-compress failed: %v\n", err)
		return false
	}
	afterTokens := a.EstimateTokens()

	// 結果を表示
	fmt.Printf("   Before: %s tokens → After: %s tokens\n",
		formatNumber(beforeTokens), formatNumber(afterTokens))
	fmt.Println("   💡 Disable with: xelyon config set compression.auto_compress false")
	fmt.Println()

	return true
}

// checkTokenWarning はトークン使用率をチェックして警告を表示
// 自動圧縮が有効な場合は表示しない（自動圧縮が処理するため）
func (a *Agent) checkTokenWarning() {
	cfg := config.GetGlobalConfig()

	// 自動圧縮が有効な場合は警告をスキップ（自動圧縮が処理する）
	if cfg.Compression.AutoCompress {
		return
	}

	percentage := a.GetTokenUsagePercentage()

	if percentage > 90 {
		yellow.Println("⚠️ Token usage is at 90%. Consider using /compress")
	} else if percentage > 80 {
		fmt.Println("💡 Token usage is high (80%). /compress available if needed")
	}
}

// handleTokenLimitError はトークン上限エラー時の提案を表示
func handleTokenLimitError(err error) {
	if !token.IsTokenLimitError(err) {
		return
	}

	red.Println("❌ Token limit exceeded")
	yellow.Println("💡 Try: /compress to reduce history")
	yellow.Println("💡 Or:  /clear to start fresh")
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
		handleTokenLimitError(err)
		return false
	}

	// 通知表示
	cyan.Println("\n⚡ トークン上限到達。自動圧縮して再実行します...")

	// LLMサマリー方式で圧縮（既存のCompressHistoryを使用）
	// keepRecentはデフォルト値10を使用
	keepRecent := 10
	cfg := config.GetGlobalConfig()
	if cfg.Compression.KeepRecent > 0 {
		keepRecent = cfg.Compression.KeepRecent
	}

	// 履歴が短すぎる場合は圧縮できない
	if len(a.History) <= keepRecent {
		yellow.Println("⚠️  履歴が短すぎるため圧縮できません")
		handleTokenLimitError(err)
		return false
	}

	// 圧縮実行
	yellow.Println("🗜️  会話を圧縮中...")
	if err := a.CompressHistory(keepRecent); err != nil {
		red.Printf("❌ 自動圧縮に失敗しました: %v\n", err)
		handleTokenLimitError(err)
		return false
	}

	// リトライカウントをインクリメント
	a.tokenLimitRetryCount++

	// リトライ実行
	cyan.Println("🔄 圧縮完了、再実行します...")
	if retryErr := retryFunc(); retryErr != nil {
		// リトライ後もエラーの場合は通常のエラー表示
		if token.IsTokenLimitError(retryErr) {
			red.Println("❌ 圧縮後もトークン上限を超えています")
		}
		handleTokenLimitError(retryErr)
		return false
	}

	// リトライ成功
	green.Println("✅ 自動圧縮＆リトライ成功")
	a.tokenLimitRetryCount = 0 // 成功したらリセット
	return true
}

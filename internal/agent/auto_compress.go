package agent

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// maybeAutoCompress は閾値を超えた場合に自動圧縮を実行
// 圧縮した場合は true を返す
func (a *Agent) maybeAutoCompress() bool {
	cfg := config.GetGlobalConfig()
	if !cfg.Compression.AutoCompress {
		return false
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

	// 圧縮実行（通知付き）
	cyan.Printf("\n🗜️ Auto-compressing history (%.0f%% threshold reached)...\n", percentage)

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
	if !isTokenLimitError(err) {
		return
	}

	red.Println("❌ Token limit exceeded")
	yellow.Println("💡 Try: /compress to reduce history")
	yellow.Println("💡 Or:  /clear to start fresh")
}

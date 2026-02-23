package gemini

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ctxKey はキャッシュリトライの context key
type ctxKey string

const cacheRetryKey ctxKey = "gemini_cache_retry"

// isCacheExpiredError はエラーがキャッシュ期限切れによるものか判定
func isCacheExpiredError(statusCode int, body []byte) bool {
	if statusCode == 404 || statusCode == 400 {
		s := string(body)
		return strings.Contains(s, "cachedContent") ||
			strings.Contains(s, "NOT_FOUND") ||
			strings.Contains(s, "not found")
	}
	return false
}

// invalidateCache はローカルのキャッシュ状態をクリアする
func (p *Provider) invalidateCache() {
	p.activeCacheName = ""
	p.cachedTokenCount = 0
	p.cachedMessageCount = 0
	p.cacheExpireTime = time.Time{}
}

// ClearCache はプロバイダーが保持するキャッシュ（リモート/ローカル）をクリアする
func (p *Provider) ClearCache() {
	if p.activeCacheName == "" {
		return
	}

	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Clearing cache: %s\n", p.activeCacheName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := p.DeleteCachedContent(ctx, p.activeCacheName)
	if err != nil && debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Failed to delete remote cache: %v\n", err)
	}

	p.invalidateCache()
}

const (
	minCacheTokens    = 32768
	maxDiffMessages   = 20   // 差分メッセージ数がこれを超えたらキャッシュ再作成
	defaultCacheTTL   = 1800 // デフォルトキャッシュTTL（秒）= 30分
	tokenEstimateRate = 1.0  // 1文字あたりのトークン概算（日本語含む）
)

// getCacheTTL はキャッシュTTL秒数を返す
// 環境変数 GEMINI_CACHE_TTL があればそちらを優先、なければデフォルト1800秒
func getCacheTTL() int {
	if v := os.Getenv("GEMINI_CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultCacheTTL
}

// estimateTokens はトークン数を概算する
func estimateTokens(systemPrompt string, history []api.Message) int {
	totalChars := len(systemPrompt)
	for _, msg := range history {
		totalChars += len(msg.Content)
	}
	return int(float64(totalChars) * tokenEstimateRate)
}

// updateOrUseCache はキャッシュの状態を確認し、更新または利用する
// tools, toolConfig: キャッシュに含めるツール定義（nilの場合は含めない）
// 戻り値:
//
//	cachedContentName: 使用するキャッシュのリソース名（空ならキャッシュなし）
//	messagesToSend: APIに送信すべきメッセージ（キャッシュ利用時は差分のみ）
//	err: エラー
func (p *Provider) updateOrUseCache(ctx context.Context, systemPrompt string, history []api.Message, model string, tools []api.GeminiToolConfig, toolConfig *GeminiToolConfigWrapper) (string, []api.Message, error) {
	// キャッシュ機能が無効化されている場合は何もしない
	if os.Getenv("GEMINI_CONTEXT_CACHING") == "0" {
		return "", history, nil
	}

	// 現在の総トークン数を概算
	totalTokens := estimateTokens(systemPrompt, history)

	// 最小トークン数未満ならキャッシュしない
	if totalTokens < minCacheTokens {
		// すでにキャッシュがある場合は削除しておく（無駄な課金を避ける）
		if p.activeCacheName != "" {
			_ = p.DeleteCachedContent(ctx, p.activeCacheName)
			p.activeCacheName = ""
			p.cachedTokenCount = 0
			p.cachedMessageCount = 0
		}
		return "", history, nil
	}

	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	// キャッシュ利用判定
	useExistingCache := false
	if p.activeCacheName != "" && time.Now().Before(p.cacheExpireTime) {
		// 前提: 履歴は追記型であること。
		// 履歴の長さが前回キャッシュ時以上で、差分が許容範囲内なら利用
		if len(history) >= p.cachedMessageCount {
			diffCount := len(history) - p.cachedMessageCount
			if diffCount <= maxDiffMessages {
				useExistingCache = true
			}
		}
	}

	if useExistingCache {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Using existing cache: %s (diff: %d msgs)\n", p.activeCacheName, len(history)-p.cachedMessageCount)
		}
		// 差分のみを返す
		diffMessages := history[p.cachedMessageCount:]
		return p.activeCacheName, diffMessages, nil
	}

	// 新規作成または再作成
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Creating new cache (tokens: ~%d)\n", totalTokens)
	}

	// 既存キャッシュがあれば削除
	if p.activeCacheName != "" {
		_ = p.DeleteCachedContent(ctx, p.activeCacheName)
		p.activeCacheName = ""
	}

	// スピナー表示
	spinner := ui.GetGlobalSpinner()
	if spinner != nil {
		spinner.SetStatus("Creating context cache...")
	}

	// generateContent には空ではない contents が必要。
	// そのため、最後のメッセージ（通常はユーザーの入力）はキャッシュに含めず、
	// generateContent で送信するようにする。
	var cacheHistory []api.Message
	var messagesToSend []api.Message

	if len(history) > 0 {
		// 最後の1つを残してキャッシュする
		cacheHistory = history[:len(history)-1]
		messagesToSend = history[len(history)-1:]
	} else {
		// 履歴がない場合はキャッシュ作成不可（システムプロンプトだけキャッシュしても意味は薄い＆呼び出し元で困る）
		return "", history, nil
	}

	// キャッシュ作成（ツール定義もキャッシュに含める）
	ttl := getCacheTTL()
	ttlStr := fmt.Sprintf("%ds", ttl)
	resp, err := p.CreateCachedContent(ctx, model, systemPrompt, cacheHistory, ttlStr, tools, toolConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create cache: %v. Proceeding without cache.\n", err)
		return "", history, nil
	}

	p.activeCacheName = resp.Name
	p.cachedTokenCount = totalTokens
	p.cachedMessageCount = len(cacheHistory)
	// 有効期限を設定（TTLの90%で期限切れ判定）
	p.cacheExpireTime = time.Now().Add(time.Duration(ttl) * time.Second * 9 / 10)

	// ストレージ料金を概算して通知
	if p.usageCallback != nil {
		ttlHours := float64(ttl) / 3600.0
		storageCost := float64(totalTokens) / 1_000_000.0 * 4.50 * ttlHours
		p.usageCallback(api.Usage{StorageCost: storageCost})
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Cache created: %s\n", resp.Name)
	}

	return p.activeCacheName, messagesToSend, nil
}

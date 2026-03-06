package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// cacheEntry はモデル別キャッシュの状態を保持する
type cacheEntry struct {
	name         string    // Gemini API のキャッシュリソース名 (e.g. "cachedContents/xxx")
	model        string    // キャッシュ作成時のモデル名
	tokenCount   int       // キャッシュされたトークン数（概算）
	messageCount int       // キャッシュに含まれるメッセージ数
	expireTime   time.Time // キャッシュの有効期限
}

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

// initCacheMap は cacheMap を lazy init する
func (p *Provider) initCacheMap() {
	if p.cacheMap == nil {
		p.cacheMap = make(map[string]*cacheEntry)
	}
}

// invalidateCache は全モデルのローカルキャッシュ状態をクリアする
func (p *Provider) invalidateCache() {
	p.cacheMap = make(map[string]*cacheEntry)
}

// invalidateCacheForModel は特定モデルのローカルキャッシュ状態をクリアする
func (p *Provider) invalidateCacheForModel(model string) {
	if p.cacheMap != nil {
		delete(p.cacheMap, model)
	}
}

// ClearCache はプロバイダーが保持するキャッシュ（リモート/ローカル）を全てクリアする
func (p *Provider) ClearCache() {
	if len(p.cacheMap) == 0 {
		return
	}

	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for model, entry := range p.cacheMap {
		if entry.name == "" {
			continue
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Clearing cache for %s: %s\n", model, entry.name)
		}
		err := p.DeleteCachedContent(ctx, entry.name)
		if err != nil && debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Failed to delete remote cache: %v\n", err)
		}
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

// estimateTokens はトークン数を概算する（ツール定義を含む）
func estimateTokens(systemPrompt string, history []api.Message, tools []api.GeminiToolConfig) int {
	totalChars := len(systemPrompt)
	for _, msg := range history {
		totalChars += len(msg.Content)
	}
	// ツール定義のトークン数を概算（JSON構造分を加算）
	for _, tool := range tools {
		for _, fd := range tool.FunctionDeclarations {
			totalChars += len(fd.Name) + len(fd.Description)
			if fd.Parameters != nil {
				paramBytes, _ := json.Marshal(fd.Parameters)
				totalChars += len(paramBytes)
			}
		}
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

	p.initCacheMap()

	// 現在の総トークン数を概算
	totalTokens := estimateTokens(systemPrompt, history, tools)

	// 最小トークン数未満ならキャッシュしない
	if totalTokens < minCacheTokens {
		// このモデルのキャッシュがある場合は削除しておく（無駄な課金を避ける）
		if entry, ok := p.cacheMap[model]; ok && entry.name != "" {
			_ = p.DeleteCachedContent(ctx, entry.name)
			delete(p.cacheMap, model)
		}
		return "", history, nil
	}

	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	// このモデルのキャッシュを取得
	entry := p.cacheMap[model]

	// キャッシュ利用判定
	useExistingCache := false
	if entry != nil && entry.name != "" && time.Now().Before(entry.expireTime) {
		// 前提: 履歴は追記型であること。
		// 履歴の長さが前回キャッシュ時以上で、差分が許容範囲内なら利用
		if len(history) >= entry.messageCount {
			diffCount := len(history) - entry.messageCount
			if diffCount <= maxDiffMessages {
				useExistingCache = true
			}
		}
	}

	if useExistingCache {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Using existing cache for %s: %s (diff: %d msgs)\n", model, entry.name, len(history)-entry.messageCount)
		}
		// 差分のみを返す
		diffMessages := history[entry.messageCount:]
		return entry.name, diffMessages, nil
	}

	// 新規作成または再作成
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Creating new cache for %s (tokens: ~%d)\n", model, totalTokens)
	}

	// このモデルの既存キャッシュがあれば削除
	if entry != nil && entry.name != "" {
		_ = p.DeleteCachedContent(ctx, entry.name)
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
		// 履歴がない場合はキャッシュ作成不可
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

	// モデル別キャッシュを保存
	p.cacheMap[model] = &cacheEntry{
		name:         resp.Name,
		model:        model,
		tokenCount:   totalTokens,
		messageCount: len(cacheHistory),
		expireTime:   time.Now().Add(time.Duration(ttl) * time.Second * 9 / 10),
	}

	// ストレージ料金を概算して通知
	if p.usageCallback != nil {
		ttlHours := float64(ttl) / 3600.0
		storageCost := float64(totalTokens) / 1_000_000.0 * 4.50 * ttlHours
		p.usageCallback(api.Usage{StorageCost: storageCost})
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG Gemini] Cache created for %s: %s\n", model, resp.Name)
	}

	return resp.Name, messagesToSend, nil
}

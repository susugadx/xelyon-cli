package gemini

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
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

const cacheNamespaceSeparator = "\x00"

func cacheMapKey(ctx context.Context, model string) string {
	namespace := api.ProviderCacheNamespaceFromContext(ctx)
	if namespace == "" {
		return model
	}
	return model + cacheNamespaceSeparator + namespace
}

func cacheDebugLabel(key string, entry *cacheEntry) string {
	if entry != nil && entry.model != "" {
		return entry.model
	}
	return key
}

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
		for key, entry := range p.cacheMap {
			if key == model || (entry != nil && entry.model == model) {
				delete(p.cacheMap, key)
			}
		}
	}
}

// invalidateCacheForRequest は現在の request namespace に対応するローカルキャッシュだけをクリアする。
func (p *Provider) invalidateCacheForRequest(ctx context.Context, model string) {
	if p.cacheMap != nil {
		delete(p.cacheMap, cacheMapKey(ctx, model))
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

	errOut := api.RuntimeOrDefault(p.runtime).ErrorOutput()
	for key, entry := range p.cacheMap {
		if entry.name == "" {
			continue
		}
		if debug {
			fmt.Fprintf(errOut, "[DEBUG Gemini] Clearing cache for %s: %s\n", cacheDebugLabel(key, entry), entry.name)
		}
		err := p.DeleteCachedContent(ctx, entry.name)
		if err != nil && debug {
			fmt.Fprintf(errOut, "[DEBUG Gemini] Failed to delete remote cache: %v\n", err)
		}
	}

	p.invalidateCache()
}

const (
	minCacheTokens  = 4096
	maxDiffMessages = 200  // 差分メッセージ数がこれを超えたらキャッシュ再作成
	defaultCacheTTL = 3600 // デフォルトキャッシュTTL（秒）= 1時間
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
func estimateTokens(model string, systemPrompt string, history []api.Message, tools []api.GeminiToolConfig) int {
	total := token.EstimateTokenCountForModel(model, systemPrompt)
	for _, msg := range history {
		total += token.EstimateTokenCountForModel(model, msg.Content)
	}
	for _, tool := range tools {
		total += token.EstimateStructuredValueTokenCountForModel(model, tool)
	}
	return total
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
	cacheKey := cacheMapKey(ctx, model)

	// 現在の総トークン数を概算
	totalTokens := estimateTokens(model, systemPrompt, history, tools)

	// 最小トークン数未満ならキャッシュしない
	if totalTokens < minCacheTokens {
		// このモデルのキャッシュがある場合は削除しておく（無駄な課金を避ける）
		if entry, ok := p.cacheMap[cacheKey]; ok && entry.name != "" {
			_ = p.DeleteCachedContent(ctx, entry.name)
			delete(p.cacheMap, cacheKey)
		}
		return "", history, nil
	}

	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	errOut := api.ErrorWriterFromContext(ctx)

	// このモデルのキャッシュを取得
	entry := p.cacheMap[cacheKey]

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
			fmt.Fprintf(errOut, "[DEBUG Gemini] Using existing cache for %s: %s (diff: %d msgs)\n", model, entry.name, len(history)-entry.messageCount)
		}
		// 差分のみを返す
		diffMessages := history[entry.messageCount:]
		return entry.name, diffMessages, nil
	}

	// 新規作成または再作成
	if debug {
		fmt.Fprintf(errOut, "[DEBUG Gemini] Creating new cache for %s (tokens: ~%d)\n", model, totalTokens)
	}

	// このモデルの既存キャッシュがあれば削除
	if entry != nil && entry.name != "" {
		_ = p.DeleteCachedContent(ctx, entry.name)
	}

	// スピナー表示
	spinner := api.SpinnerFromContext(ctx)
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
		fmt.Fprintf(api.ErrorWriterFromContext(ctx), "Warning: Failed to create cache: %v. Proceeding without cache.\n", err)
		return "", history, nil
	}

	// モデル別キャッシュを保存
	p.cacheMap[cacheKey] = &cacheEntry{
		name:         resp.Name,
		model:        model,
		tokenCount:   totalTokens,
		messageCount: len(cacheHistory),
		expireTime:   time.Now().Add(time.Duration(ttl)*time.Second - 60*time.Second),
	}

	// ストレージ料金を概算して通知
	if p.usageCallback != nil {
		estimate := cost.EstimateCacheStorageCostForConfig(config.FromContext(ctx), "gemini", model, totalTokens, ttl)
		if !estimate.PricingUnavailable && estimate.Cost > 0 {
			p.usageCallback(api.Usage{StorageCost: estimate.Cost})
		}
	}

	if debug {
		fmt.Fprintf(errOut, "[DEBUG Gemini] Cache created for %s: %s\n", model, resp.Name)
	}

	return resp.Name, messagesToSend, nil
}

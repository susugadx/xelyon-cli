package agent

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// SessionStats はセッション統計情報
type SessionStats struct {
	StartTime           time.Time
	UserMessages        int
	AssistantMessages   int
	ToolExecutions      map[string]int // ツール名 -> 実行回数
	InputTokens         int
	OutputTokens        int
	ThinkingTokens      int        // Extended Thinking トークン数（累計、出力レート課金）
	CachedInputTokens   int        // キャッシュヒットトークン数（累計）
	CacheCreationTokens int        // キャッシュ作成トークン数（累計、Claude用）
	Provider            string     // "deepseek", "openai", "claude", "gemini", "groq", "ollama"
	Model               string     // 現在のモデル名（料金計算に使用）
	LastUsage           *api.Usage // 直近のリクエストの使用量
	AccumulatedCost     float64    // リクエスト単位で計算・累積したコスト
}

// NewSessionStats は新しいSessionStatsを作成
// model は省略可能（空文字列でデフォルト料金を使用）
func NewSessionStats(provider string, model ...string) *SessionStats {
	m := ""
	if len(model) > 0 {
		m = model[0]
	}
	return &SessionStats{
		StartTime:      time.Now(),
		ToolExecutions: make(map[string]int),
		Provider:       provider,
		Model:          m,
	}
}

// AddToolExecution はツール実行をカウント
func (s *SessionStats) AddToolExecution(toolName string) {
	s.ToolExecutions[toolName]++
}

// AddTokens はトークン使用量を追加
func (s *SessionStats) AddTokens(input, output int) {
	s.InputTokens += input
	s.OutputTokens += output
}

// AddUsage は api.Usage からトークン使用量を追加
// リクエスト単位のコストを計算して AccumulatedCost に累積する
func (s *SessionStats) AddUsage(usage api.Usage) {
	s.InputTokens += usage.InputTokens
	s.OutputTokens += usage.OutputTokens
	s.ThinkingTokens += usage.ThinkingTokens
	s.CachedInputTokens += usage.CachedInputTokens
	s.CacheCreationTokens += usage.CacheCreationTokens
	s.LastUsage = &usage

	// リクエスト単位のコストを累積（Gemini 200Kティア等に対応）
	s.AccumulatedCost += CalculateRequestCostWithCache(s.Provider, s.Model, usage)
	s.AccumulatedCost += usage.StorageCost // ストレージ料金を加算
}

// TotalTokens は合計トークン数を返す
func (s *SessionStats) TotalTokens() int {
	return s.InputTokens + s.OutputTokens + s.ThinkingTokens
}

// PricingInfo はプロバイダー別の料金情報（$/1M tokens）
type PricingInfo struct {
	InputCostPerM         float64 // 通常入力トークン
	OutputCostPerM        float64 // 出力トークン
	CachedInputCostPerM   float64 // キャッシュヒット入力（割引後）
	CacheCreationCostPerM float64 // キャッシュ作成（Claude: 1.25x）
}

// GetPricingInfo はプロバイダー・モデル別の料金情報を返す
// promptTokenCount はオプション（Gemini 200Kティア判定用）
func GetPricingInfo(provider string, model string, promptTokenCount ...int) PricingInfo {
	ptc := 0
	if len(promptTokenCount) > 0 {
		ptc = promptTokenCount[0]
	}
	switch provider {
	case "deepseek":
		// DeepSeekの料金体系
		return getDeepSeekPricing(model)
	case "openai":
		return getOpenAIPricing(model)
	case "claude":
		return getClaudePricing(model)
	case "bedrock":
		if strings.Contains(strings.ToLower(model), "claude") {
			return getClaudePricing(model)
		}
		// Claude以外のBedrockモデルは一旦汎用料金
		return PricingInfo{
			InputCostPerM:         0.14,
			OutputCostPerM:        0.28,
			CachedInputCostPerM:   0.014,
			CacheCreationCostPerM: 0.14,
		}
	case "gemini":
		return getGeminiPricing(model, ptc)
	case "groq":
		return getGroqPricing(model)
	case "openrouter":
		// OpenRouterのモデル名形式: "anthropic/claude-opus-4.6", "google/gemini-3.1-pro" 等
		lm := strings.ToLower(model)
		switch {
		case strings.Contains(lm, "claude"):
			return getClaudePricing(model)
		case strings.Contains(lm, "gpt") || strings.Contains(lm, "openai") || strings.Contains(lm, "codex"):
			return getOpenAIPricing(model)
		case strings.Contains(lm, "gemini") || strings.Contains(lm, "google"):
			return getGeminiPricing(model, ptc)
		case strings.Contains(lm, "deepseek"):
			return getDeepSeekPricing(model)
		case strings.Contains(lm, "kimi") || strings.Contains(lm, "moonshotai"):
			return getKimiPricing(model)
		case strings.Contains(lm, "mistral") || strings.Contains(lm, "codestral"):
			return PricingInfo{InputCostPerM: 2.00, OutputCostPerM: 6.00, CachedInputCostPerM: 0.20, CacheCreationCostPerM: 2.00}
		case strings.Contains(lm, "llama") || strings.Contains(lm, "meta"):
			return PricingInfo{InputCostPerM: 0.20, OutputCostPerM: 0.80, CachedInputCostPerM: 0.02, CacheCreationCostPerM: 0.20}
		case strings.Contains(lm, "qwen"):
			return PricingInfo{InputCostPerM: 0.15, OutputCostPerM: 0.60, CachedInputCostPerM: 0.015, CacheCreationCostPerM: 0.15}
		default:
			return getDeepSeekPricing(model)
		}
	case "ollama":
		return PricingInfo{} // ローカル実行は無料
	default:
		// 不明なプロバイダーはDeepSeek料金で概算
		return PricingInfo{
			InputCostPerM:         0.14,
			OutputCostPerM:        0.28,
			CachedInputCostPerM:   0.014,
			CacheCreationCostPerM: 0.14,
		}
	}
}

// getDeepSeekPricing はモデル名からDeepSeek料金を返す
func getDeepSeekPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if strings.Contains(lm, "reasoner") {
		// DeepSeek Reasoner: $0.55/$2.19 per million tokens, Cache hit: $0.14
		return PricingInfo{
			InputCostPerM:         0.55,
			OutputCostPerM:        2.19,
			CachedInputCostPerM:   0.14,
			CacheCreationCostPerM: 0.55,
		}
	}
	// DeepSeek Chat（デフォルト）: $0.14/$0.28 per million tokens, Cache hit: $0.014
	return PricingInfo{
		InputCostPerM:         0.14,
		OutputCostPerM:        0.28,
		CachedInputCostPerM:   0.014,
		CacheCreationCostPerM: 0.14,
	}
}

// getClaudePricing はモデル名からClaude料金を返す
func getClaudePricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "opus"):
		// Opus 4.5/4.6: $5/$25 per million tokens
		return PricingInfo{
			InputCostPerM:         5.00,
			OutputCostPerM:        25.00,
			CachedInputCostPerM:   0.50, // 90% off
			CacheCreationCostPerM: 6.25, // 25% premium
		}
	case strings.Contains(lm, "haiku"):
		// Haiku 4.5: $1/$5 per million tokens
		return PricingInfo{
			InputCostPerM:         1.00,
			OutputCostPerM:        5.00,
			CachedInputCostPerM:   0.10, // 90% off
			CacheCreationCostPerM: 1.25, // 25% premium
		}
	default:
		// Sonnet 4.5（デフォルト）: $3/$15 per million tokens
		return PricingInfo{
			InputCostPerM:         3.00,
			OutputCostPerM:        15.00,
			CachedInputCostPerM:   0.30, // 90% off
			CacheCreationCostPerM: 3.75, // 25% premium
		}
	}
}

// getOpenAIPricing はモデル名からOpenAI料金を返す
func getOpenAIPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "nano"):
		// GPT-5 Nano: $0.05/$0.40 per million tokens
		return PricingInfo{
			InputCostPerM:         0.05,
			OutputCostPerM:        0.40,
			CachedInputCostPerM:   0.005, // 90% off
			CacheCreationCostPerM: 0.05,
		}
	case strings.Contains(lm, "mini"):
		// GPT-5 Mini / Codex-Mini: $0.25/$2.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.25,
			OutputCostPerM:        2.00,
			CachedInputCostPerM:   0.025, // 90% off
			CacheCreationCostPerM: 0.25,
		}
	case strings.Contains(lm, "5.2-pro"):
		// GPT-5.2 Pro: $21/$168 per million tokens
		return PricingInfo{
			InputCostPerM:         21.00,
			OutputCostPerM:        168.00,
			CachedInputCostPerM:   2.10, // 90% off
			CacheCreationCostPerM: 21.00,
		}
	case strings.Contains(lm, "5.1"):
		// GPT-5.1 / 5.1-Codex: $2.00/$8.00 per million tokens
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        8.00,
			CachedInputCostPerM:   0.50,
			CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "5.2"):
		// GPT-5.2 / 5.2-Codex: $1.75/$14 per million tokens
		return PricingInfo{
			InputCostPerM:         1.75,
			OutputCostPerM:        14.00,
			CachedInputCostPerM:   0.175, // 90% off
			CacheCreationCostPerM: 1.75,
		}
	default:
		// GPT-5 / 5.1 / Codex（デフォルト）: $1.25/$10 per million tokens
		return PricingInfo{
			InputCostPerM:         1.25,
			OutputCostPerM:        10.00,
			CachedInputCostPerM:   0.125, // 90% off
			CacheCreationCostPerM: 1.25,
		}
	}
}

// getGeminiPricing はモデル名からGemini料金を返す
// promptTokenCount はリクエストの入力トークン数（200Kティア判定に使用）
func getGeminiPricing(model string, promptTokenCount int) PricingInfo {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "pro"):
		if promptTokenCount > 200000 {
			// Long context pricing (>200K): $4/$18 per million tokens
			return PricingInfo{
				InputCostPerM:         4.00,
				OutputCostPerM:        18.00,
				CachedInputCostPerM:   0.40,
				CacheCreationCostPerM: 4.00,
			}
		}
		// Standard pricing (<=200K): $2/$12 per million tokens
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        12.00,
			CachedInputCostPerM:   0.20, // 90% off
			CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "2.5-flash"), strings.Contains(lm, "2.5-flash-lite"):
		// Gemini 2.5 Flash: $0.30/$2.50 per million tokens
		return PricingInfo{
			InputCostPerM:         0.30,
			OutputCostPerM:        2.50,
			CachedInputCostPerM:   0.03, // 90% off
			CacheCreationCostPerM: 0.30,
		}
	default:
		// Gemini 3 Flash / 2.0 Flash（デフォルト）: $0.50/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.50,
			OutputCostPerM:        3.00,
			CachedInputCostPerM:   0.05, // 90% off
			CacheCreationCostPerM: 0.50,
		}
	}
}

// getGroqPricing はモデル名からGroq料金を返す
func getGroqPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "70b"):
		// Llama 3/3.1 70B: $0.59/$0.79 per million tokens
		return PricingInfo{
			InputCostPerM:         0.59,
			OutputCostPerM:        0.79,
			CachedInputCostPerM:   0.59, // キャッシュ割引なし
			CacheCreationCostPerM: 0.59,
		}
	case strings.Contains(lm, "405b"):
		// Llama 3.1 405B: $2.00/$2.00 per million tokens (approximate)
		return PricingInfo{
			InputCostPerM:         2.00,
			OutputCostPerM:        2.00,
			CachedInputCostPerM:   2.00,
			CacheCreationCostPerM: 2.00,
		}
	case strings.Contains(lm, "mixtral"):
		// Mixtral 8x7B: $0.24/$0.24 per million tokens
		return PricingInfo{
			InputCostPerM:         0.24,
			OutputCostPerM:        0.24,
			CachedInputCostPerM:   0.24,
			CacheCreationCostPerM: 0.24,
		}
	case strings.Contains(lm, "gemma"):
		// Gemma 7B: $0.07/$0.07 per million tokens
		return PricingInfo{
			InputCostPerM:         0.07,
			OutputCostPerM:        0.07,
			CachedInputCostPerM:   0.07,
			CacheCreationCostPerM: 0.07,
		}
	default:
		// Llama 3/3.1 8B (default): $0.05/$0.10 per million tokens
		return PricingInfo{
			InputCostPerM:         0.05,
			OutputCostPerM:        0.10,
			CachedInputCostPerM:   0.05,
			CacheCreationCostPerM: 0.05,
		}
	}
}

// getKimiPricing はモデル名からKimi料金を返す
func getKimiPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	if strings.Contains(lm, "k2.5") {
		// Kimi K2.5: $0.60/$3.00 per million tokens
		return PricingInfo{
			InputCostPerM:         0.60,
			OutputCostPerM:        3.00,
			CachedInputCostPerM:   0.06,
			CacheCreationCostPerM: 0.60,
		}
	}
	// Kimi K2（デフォルト）: $0.60/$2.50 per million tokens
	return PricingInfo{
		InputCostPerM:         0.60,
		OutputCostPerM:        2.50,
		CachedInputCostPerM:   0.06,
		CacheCreationCostPerM: 0.60,
	}
}

// EstimatedCost は推定コストを計算（USD）
// AddUsage 経由で累積されたコストがあればそれを使い、
// AddTokens() のみ使われた場合はフォールバック計算を行う
func (s *SessionStats) EstimatedCost() float64 {
	if s.Provider == "ollama" {
		return 0.0 // ローカル実行
	}

	// AddUsage 経由で累積されたコストがあればそれを使う
	if s.AccumulatedCost > 0 {
		return s.AccumulatedCost
	}

	// フォールバック: AddTokens() のみ使われた場合（レガシー互換）
	pricing := GetPricingInfo(s.Provider, s.Model)

	cachedInputCost := float64(s.CachedInputTokens) / 1_000_000.0 * pricing.CachedInputCostPerM
	cacheCreationCost := float64(s.CacheCreationTokens) / 1_000_000.0 * pricing.CacheCreationCostPerM

	uncachedInput := s.InputTokens - s.CachedInputTokens - s.CacheCreationTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	uncachedInputCost := float64(uncachedInput) / 1_000_000.0 * pricing.InputCostPerM

	outputCost := float64(s.OutputTokens) / 1_000_000.0 * pricing.OutputCostPerM
	thinkingCost := float64(s.ThinkingTokens) / 1_000_000.0 * pricing.OutputCostPerM

	return cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost + thinkingCost
}

// ElapsedTime はセッション開始からの経過時間を返す
func (s *SessionStats) ElapsedTime() time.Duration {
	return time.Since(s.StartTime)
}

// FormatElapsedTime は経過時間を人間が読める形式で返す
func (s *SessionStats) FormatElapsedTime() string {
	elapsed := s.ElapsedTime()
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// TotalMessages は合計メッセージ数を返す
func (s *SessionStats) TotalMessages() int {
	return s.UserMessages + s.AssistantMessages
}

// TotalToolExecutions は合計ツール実行回数を返す
func (s *SessionStats) TotalToolExecutions() int {
	total := 0
	for _, count := range s.ToolExecutions {
		total += count
	}
	return total
}

// GetSessionFileSize はセッションファイルのサイズを返す（bytes）
func GetSessionFileSize(sessionPath string) (int64, error) {
	info, err := os.Stat(sessionPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// FormatFileSize はファイルサイズを人間が読める形式で返す
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatTokens は K/M 形式でトークン数をフォーマット
func FormatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// FormatNumber は数値にカンマを追加してフォーマット
func FormatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	// 再帰的にカンマを追加（10000以上も対応）
	return FormatNumber(n/1000) + fmt.Sprintf(",%03d", n%1000)
}

// CalculateRequestCost は単一リクエストのコストを計算（キャッシュなし想定）
func CalculateRequestCost(provider, model string, input, output int) float64 {
	if provider == "ollama" {
		return 0.0 // ローカル実行
	}

	pricing := GetPricingInfo(provider, model, input)

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(input) / 1_000_000.0) * pricing.InputCostPerM
	outputCostUSD := (float64(output) / 1_000_000.0) * pricing.OutputCostPerM

	return inputCostUSD + outputCostUSD
}

// CalculateRequestCostWithCache は単一リクエストのコストを計算（キャッシュ対応）
func CalculateRequestCostWithCache(provider, model string, usage api.Usage) float64 {
	if provider == "ollama" {
		return 0.0
	}

	pricing := GetPricingInfo(provider, model, usage.InputTokens)

	cachedInputCost := float64(usage.CachedInputTokens) / 1_000_000.0 * pricing.CachedInputCostPerM
	cacheCreationCost := float64(usage.CacheCreationTokens) / 1_000_000.0 * pricing.CacheCreationCostPerM

	uncachedInput := usage.InputTokens - usage.CachedInputTokens - usage.CacheCreationTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	uncachedInputCost := float64(uncachedInput) / 1_000_000.0 * pricing.InputCostPerM

	outputCost := float64(usage.OutputTokens) / 1_000_000.0 * pricing.OutputCostPerM

	// Thinking tokens は出力レートで課金（candidatesTokenCount とは別計上）
	thinkingCost := float64(usage.ThinkingTokens) / 1_000_000.0 * pricing.OutputCostPerM

	return cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost + thinkingCost
}

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
	CachedInputTokens   int        // キャッシュヒットトークン数（累計）
	CacheCreationTokens int        // キャッシュ作成トークン数（累計、Claude用）
	Provider            string     // "deepseek", "openai", "claude", "gemini", "groq", "ollama"
	Model               string     // 現在のモデル名（料金計算に使用）
	LastUsage           *api.Usage // 直近のリクエストの使用量
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
func (s *SessionStats) AddUsage(usage api.Usage) {
	s.InputTokens += usage.InputTokens
	s.OutputTokens += usage.OutputTokens
	s.CachedInputTokens += usage.CachedInputTokens
	s.CacheCreationTokens += usage.CacheCreationTokens
	s.LastUsage = &usage
}

// TotalTokens は合計トークン数を返す
func (s *SessionStats) TotalTokens() int {
	return s.InputTokens + s.OutputTokens
}

// PricingInfo はプロバイダー別の料金情報（$/1M tokens）
type PricingInfo struct {
	InputCostPerM         float64 // 通常入力トークン
	OutputCostPerM        float64 // 出力トークン
	CachedInputCostPerM   float64 // キャッシュヒット入力（割引後）
	CacheCreationCostPerM float64 // キャッシュ作成（Claude: 1.25x）
}

// GetPricingInfo はプロバイダー・モデル別の料金情報を返す
func GetPricingInfo(provider string, model string) PricingInfo {
	switch provider {
	case "deepseek":
		// DeepSeek: キャッシュヒット90%割引 ($0.14 → $0.014)
		return PricingInfo{
			InputCostPerM:         0.14,
			OutputCostPerM:        0.28,
			CachedInputCostPerM:   0.014, // 90% off
			CacheCreationCostPerM: 0.14,  // 通常料金
		}
	case "openai":
		return getOpenAIPricing(model)
	case "claude", "bedrock":
		return getClaudePricing(model)
	case "gemini":
		return getGeminiPricing(model)
	case "groq":
		return PricingInfo{
			InputCostPerM:         0.10,
			OutputCostPerM:        0.10,
			CachedInputCostPerM:   0.10, // キャッシュなし
			CacheCreationCostPerM: 0.10,
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
	case strings.Contains(lm, "5.2"):
		// GPT-5.2 / 5.2-Codex: $1.75/$14 per million tokens
		return PricingInfo{
			InputCostPerM:         1.75,
			OutputCostPerM:        14.00,
			CachedInputCostPerM:   0.18, // 90% off
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
func getGeminiPricing(model string) PricingInfo {
	lm := strings.ToLower(model)
	switch {
	case strings.Contains(lm, "pro"):
		// Gemini 3 Pro / 2.5 Pro: $2/$12 per million tokens
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

// EstimatedCost は推定コストを計算（USD）
// キャッシュヒットとキャッシュ作成の割引/割増を反映
func (s *SessionStats) EstimatedCost() float64 {
	if s.Provider == "ollama" {
		return 0.0 // ローカル実行
	}

	pricing := GetPricingInfo(s.Provider, s.Model)

	// 入力トークンのコスト計算
	// - CachedInputTokens: キャッシュヒット（割引適用）
	// - CacheCreationTokens: キャッシュ作成（Claude: 割増）
	// - 残り: 通常入力トークン
	cachedInputCost := float64(s.CachedInputTokens) / 1_000_000.0 * pricing.CachedInputCostPerM
	cacheCreationCost := float64(s.CacheCreationTokens) / 1_000_000.0 * pricing.CacheCreationCostPerM

	// 通常入力トークン = 全入力 - キャッシュヒット - キャッシュ作成
	// 注: InputTokens は API から返される総入力トークン数
	// キャッシュの場合、InputTokens = CachedInputTokens + CacheCreationTokens + 通常入力 となる
	uncachedInput := s.InputTokens - s.CachedInputTokens - s.CacheCreationTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	uncachedInputCost := float64(uncachedInput) / 1_000_000.0 * pricing.InputCostPerM

	// 出力トークンのコスト
	outputCost := float64(s.OutputTokens) / 1_000_000.0 * pricing.OutputCostPerM

	return cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost
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

	pricing := GetPricingInfo(provider, model)

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

	pricing := GetPricingInfo(provider, model)

	cachedInputCost := float64(usage.CachedInputTokens) / 1_000_000.0 * pricing.CachedInputCostPerM
	cacheCreationCost := float64(usage.CacheCreationTokens) / 1_000_000.0 * pricing.CacheCreationCostPerM

	uncachedInput := usage.InputTokens - usage.CachedInputTokens - usage.CacheCreationTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	uncachedInputCost := float64(uncachedInput) / 1_000_000.0 * pricing.InputCostPerM

	outputCost := float64(usage.OutputTokens) / 1_000_000.0 * pricing.OutputCostPerM

	return cachedInputCost + cacheCreationCost + uncachedInputCost + outputCost
}

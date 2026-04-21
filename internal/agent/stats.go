package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// SavingsMetrics はコスト最適化による削減量の推定値を保持する。
// 値はすべて推定（estimated）であり、表示時は「~」を付与する。
type SavingsMetrics struct {
	SavedCalls                int     // 省略・統合されたツール呼び出し数
	EstimatedInputTokensSaved int     // 推定入力トークン削減量
	EstimatedCostSaved        float64 // 推定コスト削減量（USD）
}

func (m *SavingsMetrics) add(other SavingsMetrics) {
	m.SavedCalls += other.SavedCalls
	m.EstimatedInputTokensSaved += other.EstimatedInputTokensSaved
	m.EstimatedCostSaved += other.EstimatedCostSaved
}

func (m *SavingsMetrics) hasAny() bool {
	return m.SavedCalls > 0 || m.EstimatedInputTokensSaved > 0 || m.EstimatedCostSaved > 0
}

// OptimizationMetrics は最適化機構の発動回数を保持する。
type OptimizationMetrics struct {
	NegativeCacheHits      int // ネガティブキャッシュヒット
	ErrorCompressions      int // compressErrorResult 発動
	FailedPairCompressions int // compressFailedPair 発動
	OutlineFirstCount      int // outline-first mode発動
	CompactionCount        int // auto-compress実行
	CostAwareCompressions  int // pricing cliff回避による auto-compress 実行
}

func (m *OptimizationMetrics) add(other OptimizationMetrics) {
	m.NegativeCacheHits += other.NegativeCacheHits
	m.ErrorCompressions += other.ErrorCompressions
	m.FailedPairCompressions += other.FailedPairCompressions
	m.OutlineFirstCount += other.OutlineFirstCount
	m.CompactionCount += other.CompactionCount
	m.CostAwareCompressions += other.CostAwareCompressions
}

func (m *OptimizationMetrics) addCompaction(other CompactionMetrics) {
	m.ErrorCompressions += other.ErrorCompressions
	m.FailedPairCompressions += other.FailedPairCompressions
}

func (m *OptimizationMetrics) hasAny() bool {
	return m.NegativeCacheHits > 0 ||
		m.ErrorCompressions > 0 ||
		m.FailedPairCompressions > 0 ||
		m.OutlineFirstCount > 0 ||
		m.CompactionCount > 0 ||
		m.CostAwareCompressions > 0
}

// ToolObservability はツール実行・compaction の観測メトリクスを保持する。
type ToolObservability struct {
	ReadFileEmptyPathsErrors     int // read_file が paths is empty で失敗した回数
	ReadFileBatchCalls           int // read_file(paths=...) の batch 呼び出し回数
	ReadFileTargetCalls          int // read_file(targets=...) の呼び出し回数
	SearchCodeImpactCalls        int // search_code(intent=impact) の呼び出し回数
	SearchCodeExplicitMultiCalls int // search_code の明示的 multi-pattern 呼び出し回数
	SearchCodeMultiPatternCalls  int // search_code の multi-pattern 呼び出し回数
	SearchCodeMissedMultiPattern int // search_code の serial single-pattern から観測した missed multi-pattern 回数
	SearchCodeBatchMerges        int // search_code multi-pattern batch merge 回数
	ReadFileBatchMerges          int // read_file batch merge 回数
}

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
	LastTurnUsage       *api.Usage // 直近のユーザーリクエスト全体の使用量
	LastTurnCost        float64    // 直近のユーザーリクエスト全体の正確なコスト
	AccumulatedCost     float64    // リクエスト単位で計算・累積したコスト
	Optimizations       OptimizationMetrics
	ToolObs             ToolObservability // ツール実行・compaction の観測メトリクス
	Savings             SavingsMetrics    // コスト最適化による推定削減量
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
	// キャッシュ情報がないので InputTokens でティア判定
	totalInputForTier := s.InputTokens + s.CachedInputTokens + s.CacheCreationTokens
	pricing := GetPricingInfo(s.Provider, s.Model, totalInputForTier)

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

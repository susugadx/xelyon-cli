package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
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
	ApplyPatchAttempts           int // Gemini apply_patch 実行試行回数
	ApplyPatchSuccesses          int // Gemini apply_patch 成功回数（repair 成功を含む）
	ApplyPatchRepairAttempts     int // Gemini apply_patch repair 試行回数
	ApplyPatchRepairSuccesses    int // Gemini apply_patch repair 成功回数
}

// SessionStats はセッション統計情報
type SessionStats struct {
	StartTime             time.Time
	UserMessages          int
	AssistantMessages     int
	ToolExecutions        map[string]int // ツール名 -> 実行回数
	InputTokens           int
	OutputTokens          int
	ThinkingTokens        int        // Extended Thinking トークン数（累計、出力レート課金）
	CachedInputTokens     int        // キャッシュヒットトークン数（累計）
	CacheCreationTokens   int        // キャッシュ作成トークン数（累計、Claude用）
	WebSearchCalls        int        // built-in web search 呼び出し回数（累計）
	WebSearchResultTokens int        // provider が返した検索結果 token 観測値（累計、入力 tokens には再加算しない）
	WebSearchCost         float64    // built-in web search 固定料金（USD、累計）
	Provider              string     // "deepseek", "openai", "claude", "gemini", "groq", "ollama"
	Model                 string     // 現在のモデル名（料金計算に使用）
	LastUsage             *api.Usage // 直近のリクエストの使用量
	LastTurnUsage         *api.Usage // 直近のユーザーリクエスト全体の使用量
	LastTurnCost          float64    // 直近のユーザーリクエスト全体の正確なコスト
	LastTurnCostUnknown   bool       // 直近ターンに既知の料金表がない provider/model が含まれる
	AccumulatedCost       float64    // リクエスト単位で計算・累積したコスト
	CostUnknown           bool       // セッション累積に既知の料金表がない provider/model が含まれる
	CostUnknownEvents     int        // 料金不明のリクエスト累計数
	Optimizations         OptimizationMetrics
	ToolObs               ToolObservability // ツール実行・compaction の観測メトリクス
	Savings               SavingsMetrics    // コスト最適化による推定削減量
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
	s.AddUsageForConfig(nil, usage)
}

// AddUsageForConfig は catalog_model 設定を考慮して api.Usage を累積する。
func (s *SessionStats) AddUsageForConfig(cfg *config.Config, usage api.Usage) {
	s.AddUsageForProviderConfig(cfg, s.Provider, s.Model, usage)
}

// AddUsageForProviderConfig は指定 request owner の料金表で api.Usage を累積する。
func (s *SessionStats) AddUsageForProviderConfig(cfg *config.Config, provider, model string, usage api.Usage) {
	s.InputTokens += usage.InputTokens
	s.OutputTokens += usage.OutputTokens
	s.ThinkingTokens += usage.ThinkingTokens
	s.CachedInputTokens += usage.CachedInputTokens
	s.CacheCreationTokens += usage.CacheCreationTokens
	s.WebSearchCalls += usage.WebSearchCalls
	s.WebSearchResultTokens += usage.WebSearchResultTokens
	if usage.WebSearchCalls > 0 {
		s.WebSearchCost += usage.StorageCost
	}
	s.LastUsage = &usage

	// リクエスト単位のコストを累積（Gemini 200Kティア等に対応）
	estimate := cost.EstimateRequestCostWithCacheForConfig(cfg, provider, model, usage)
	s.AccumulatedCost += estimate.Cost
	s.AccumulatedCost += usage.StorageCost // トークン料金とは別枠の固定料金を加算
	if estimate.PricingUnavailable {
		s.CostUnknown = true
		s.CostUnknownEvents++
	}
}

// UsageDeltaSince は指定時点から現在までの request usage 相当の差分を返す。
func (s *SessionStats) UsageDeltaSince(start SessionStats) api.Usage {
	if s == nil {
		return api.Usage{}
	}
	return api.Usage{
		InputTokens:           s.InputTokens - start.InputTokens,
		OutputTokens:          s.OutputTokens - start.OutputTokens,
		ThinkingTokens:        s.ThinkingTokens - start.ThinkingTokens,
		CachedInputTokens:     s.CachedInputTokens - start.CachedInputTokens,
		CacheCreationTokens:   s.CacheCreationTokens - start.CacheCreationTokens,
		StorageCost:           s.WebSearchCost - start.WebSearchCost,
		WebSearchCalls:        s.WebSearchCalls - start.WebSearchCalls,
		WebSearchResultTokens: s.WebSearchResultTokens - start.WebSearchResultTokens,
	}
}

func (s *SessionStats) ResetUsageForProvider(provider, model string) {
	s.Provider = provider
	s.Model = model
	s.InputTokens = 0
	s.CachedInputTokens = 0
	s.CacheCreationTokens = 0
	s.WebSearchCalls = 0
	s.WebSearchResultTokens = 0
	s.WebSearchCost = 0
	s.OutputTokens = 0
	s.ThinkingTokens = 0
	s.ToolExecutions = make(map[string]int)
	s.LastUsage = nil
	s.LastTurnUsage = nil
	s.LastTurnCost = 0
	s.LastTurnCostUnknown = false
	s.AccumulatedCost = 0
	s.CostUnknown = false
	s.CostUnknownEvents = 0
}

// TotalTokens は合計トークン数を返す
func (s *SessionStats) TotalTokens() int {
	return s.InputTokens + s.OutputTokens + s.ThinkingTokens
}

// EstimatedCost は推定コストを計算（USD）
// AddUsage 経由で累積されたコストがあればそれを使い、
// AddTokens() のみ使われた場合は usage 相当の推定計算を行う。
func (s *SessionStats) EstimatedCost() float64 {
	return s.EstimatedCostForConfig(nil)
}

// EstimatedCostForConfig は catalog_model 設定を考慮して推定コストを計算する。
func (s *SessionStats) EstimatedCostForConfig(cfg *config.Config) float64 {
	return s.EstimatedCostEstimateForConfig(cfg).Cost
}

func (s *SessionStats) EstimatedCostEstimateForConfig(cfg *config.Config) cost.CostEstimate {
	// AddUsage 経由で累積されたコストがあればそれを使う
	if s.AccumulatedCost > 0 || s.CostUnknown {
		return cost.CostEstimate{
			Cost:               s.AccumulatedCost,
			PricingUnavailable: s.CostUnknown,
		}
	}

	if s.Provider == "ollama" {
		return cost.CostEstimate{} // ローカル実行
	}

	// AddTokens() のみ使われた場合（レガシー互換）
	// キャッシュ情報がないので InputTokens でティア判定
	return cost.EstimateRequestCostWithCacheForConfig(cfg, s.Provider, s.Model, api.Usage{
		InputTokens:         s.InputTokens,
		OutputTokens:        s.OutputTokens,
		ThinkingTokens:      s.ThinkingTokens,
		CachedInputTokens:   s.CachedInputTokens,
		CacheCreationTokens: s.CacheCreationTokens,
	})
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

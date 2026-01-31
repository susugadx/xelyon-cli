package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// SessionStats はセッション統計情報
type SessionStats struct {
	StartTime         time.Time
	UserMessages      int
	AssistantMessages int
	ToolExecutions    map[string]int // ツール名 -> 実行回数
	InputTokens       int
	OutputTokens      int
	Provider          string     // "deepseek", "openai", "claude", "gemini", "groq", "ollama"
	LastUsage         *api.Usage // 直近のリクエストの使用量
}

// NewSessionStats は新しいSessionStatsを作成
func NewSessionStats(provider string) *SessionStats {
	return &SessionStats{
		StartTime:      time.Now(),
		ToolExecutions: make(map[string]int),
		Provider:       provider,
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

// TotalTokens は合計トークン数を返す
func (s *SessionStats) TotalTokens() int {
	return s.InputTokens + s.OutputTokens
}

// EstimatedCost は推定コストを計算（USD）
func (s *SessionStats) EstimatedCost() float64 {
	if s.Provider == "ollama" {
		return 0.0 // ローカル実行
	}

	// 1M tokens あたりの料金（USD）
	var inputCost, outputCost float64
	switch s.Provider {
	case "deepseek":
		inputCost = 0.14
		outputCost = 0.28
	case "openai":
		inputCost = 2.50
		outputCost = 10.00
	case "claude":
		inputCost = 3.00
		outputCost = 15.00
	case "gemini":
		inputCost = 0.075
		outputCost = 0.30
	case "groq":
		// Groqは無料枠があるが、有料プランの料金で計算
		inputCost = 0.10
		outputCost = 0.10
	default:
		// 不明なプロバイダーはDeepSeek料金で概算
		inputCost = 0.14
		outputCost = 0.28
	}

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(s.InputTokens) / 1_000_000.0) * inputCost
	outputCostUSD := (float64(s.OutputTokens) / 1_000_000.0) * outputCost

	return inputCostUSD + outputCostUSD
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
	return fmt.Sprintf("%d,%03d", n/1000, n%1000)
}

// CalculateRequestCost は単一リクエストのコストを計算
func CalculateRequestCost(provider string, input, output int) float64 {
	if provider == "ollama" {
		return 0.0 // ローカル実行
	}

	// 1M tokens あたりの料金（USD）
	var inputCost, outputCost float64
	switch provider {
	case "deepseek":
		inputCost = 0.14
		outputCost = 0.28
	case "openai":
		inputCost = 2.50
		outputCost = 10.00
	case "claude":
		inputCost = 3.00
		outputCost = 15.00
	case "gemini":
		inputCost = 0.075
		outputCost = 0.30
	case "groq":
		inputCost = 0.10
		outputCost = 0.10
	default:
		inputCost = 0.14
		outputCost = 0.28
	}

	// コスト計算: (tokens / 1,000,000) * price
	inputCostUSD := (float64(input) / 1_000_000.0) * inputCost
	outputCostUSD := (float64(output) / 1_000_000.0) * outputCost

	return inputCostUSD + outputCostUSD
}

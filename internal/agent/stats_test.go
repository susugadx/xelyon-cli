package agent

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNewSessionStats(t *testing.T) {
	stats := NewSessionStats("deepseek")

	if stats.Provider != "deepseek" {
		t.Errorf("NewSessionStats() Provider = %v, want 'deepseek'", stats.Provider)
	}

	if stats.ToolExecutions == nil {
		t.Error("NewSessionStats() ToolExecutions should not be nil")
	}

	if stats.UserMessages != 0 {
		t.Errorf("NewSessionStats() UserMessages = %d, want 0", stats.UserMessages)
	}

	if stats.StartTime.IsZero() {
		t.Error("NewSessionStats() StartTime should be set")
	}
}

func TestSessionStats_AddToolExecution(t *testing.T) {
	stats := NewSessionStats("test")

	stats.AddToolExecution("read_file")
	stats.AddToolExecution("read_file")
	stats.AddToolExecution("write_file")

	if stats.ToolExecutions["read_file"] != 2 {
		t.Errorf("AddToolExecution() read_file count = %d, want 2", stats.ToolExecutions["read_file"])
	}

	if stats.ToolExecutions["write_file"] != 1 {
		t.Errorf("AddToolExecution() write_file count = %d, want 1", stats.ToolExecutions["write_file"])
	}
}

func TestSessionStats_AddTokens(t *testing.T) {
	stats := NewSessionStats("test")

	stats.AddTokens(100, 200)
	stats.AddTokens(50, 75)

	if stats.InputTokens != 150 {
		t.Errorf("AddTokens() InputTokens = %d, want 150", stats.InputTokens)
	}

	if stats.OutputTokens != 275 {
		t.Errorf("AddTokens() OutputTokens = %d, want 275", stats.OutputTokens)
	}
}

func TestSessionStats_AddUsage(t *testing.T) {
	stats := NewSessionStats("test")

	usage := api.Usage{
		InputTokens:         100,
		OutputTokens:        200,
		CachedInputTokens:   50,
		CacheCreationTokens: 25,
		StorageCost:         1.25,
	}

	stats.AddUsage(usage)

	if stats.InputTokens != 100 {
		t.Errorf("AddUsage() InputTokens = %d, want 100", stats.InputTokens)
	}
	if stats.OutputTokens != 200 {
		t.Errorf("AddUsage() OutputTokens = %d, want 200", stats.OutputTokens)
	}
	if stats.CachedInputTokens != 50 {
		t.Errorf("AddUsage() CachedInputTokens = %d, want 50", stats.CachedInputTokens)
	}
	if stats.CacheCreationTokens != 25 {
		t.Errorf("AddUsage() CacheCreationTokens = %d, want 25", stats.CacheCreationTokens)
	}
	if stats.LastUsage == nil {
		t.Error("AddUsage() LastUsage should not be nil")
	} else if stats.LastUsage.InputTokens != 100 {
		t.Errorf("AddUsage() LastUsage.InputTokens = %d, want 100", stats.LastUsage.InputTokens)
	}

	// test プロバイダのデフォルトコスト(0.28+0.42 DeepSeek V3.2)ベースのリクエスト単位コスト計算のうえ、StorageCostが加算される
	// 100 input, 200 output, 50 cached in, 25 cache creation
	// unknown provider → DeepSeek V3.2 料金:
	//   uncachedInput: (100-50-25)/1M * 0.28 = 25 * 0.00000028 = 0.000007
	//   CachedInput: 50/1M * 0.028 = 0.0000014
	//   CacheCreation: 25/1M * 0.28 = 0.000007
	//   Output: 200/1M * 0.42 = 0.000084
	//   StorageCost: 1.25
	// 合計: ~0.0000994 + 1.25 ≈ 1.2500994
	if stats.AccumulatedCost < 1.25 || stats.AccumulatedCost > 1.26 {
		t.Errorf("AddUsage() AccumulatedCost = %f, expected around 1.25", stats.AccumulatedCost)
	}
}

func TestSessionStats_TotalTokens(t *testing.T) {
	stats := NewSessionStats("test")
	stats.AddTokens(100, 200)

	total := stats.TotalTokens()
	if total != 300 {
		t.Errorf("TotalTokens() = %d, want 300", total)
	}
}

func TestSessionStats_EstimatedCost_Ollama(t *testing.T) {
	stats := NewSessionStats("ollama")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	if cost != 0.0 {
		t.Errorf("EstimatedCost() for ollama = %f, want 0.0", cost)
	}
}

func TestSessionStats_EstimatedCost_DeepSeek(t *testing.T) {
	stats := NewSessionStats("deepseek")
	stats.AddTokens(1000000, 1000000) // 1M input, 1M output

	cost := stats.EstimatedCost()
	expected := 0.28 + 0.42 // DeepSeek V3.2: $0.28/1M + $0.42/1M

	// 浮動小数点の比較は許容誤差で
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for deepseek = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenAI(t *testing.T) {
	stats := NewSessionStats("openai")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 1.25 + 10.00
	if cost != expected {
		t.Errorf("EstimatedCost() for openai = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude(t *testing.T) {
	// デフォルト（model=""）は Sonnet 料金
	// 1M input > 200K → long context 料金 ($6/$22.50)
	stats := NewSessionStats("claude")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 6.00 + 22.50 // Sonnet long context
	if cost != expected {
		t.Errorf("EstimatedCost() for claude (sonnet long context) = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude_Opus(t *testing.T) {
	// 1M input > 200K → long context 料金 ($10/$37.50)
	stats := NewSessionStats("claude", "claude-opus-4-5-20251101")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 10.00 + 37.50 // Opus long context
	if cost != expected {
		t.Errorf("EstimatedCost() for claude opus long context = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude_Haiku(t *testing.T) {
	// Haiku は long context なし
	stats := NewSessionStats("claude", "claude-haiku-4-5-20251001")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 1.00 + 5.00
	if cost != expected {
		t.Errorf("EstimatedCost() for claude haiku = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Bedrock_Opus(t *testing.T) {
	// 1M input > 200K → long context 料金 ($10/$37.50)
	stats := NewSessionStats("bedrock", "global.anthropic.claude-opus-4-5-20251101-v1:0")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 10.00 + 37.50 // Opus long context
	if cost != expected {
		t.Errorf("EstimatedCost() for bedrock opus long context = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Gemini(t *testing.T) {
	stats := NewSessionStats("gemini")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.50 + 3.00
	if cost != expected {
		t.Errorf("EstimatedCost() for gemini = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Groq(t *testing.T) {
	stats := NewSessionStats("groq") // default model (Llama 8B)
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.05 + 0.10
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for groq = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_Claude(t *testing.T) {
	// 1M input > 200K → Sonnet long context 料金 ($6/$22.50)
	stats := NewSessionStats("openrouter", "anthropic/claude-sonnet-4.5")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 6.00 + 22.50 // Sonnet long context pricing
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter claude = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_Gemini(t *testing.T) {
	// 1M input > 200K → Gemini 3.1 Pro long context ($4/$18)
	stats := NewSessionStats("openrouter", "google/gemini-3.1-pro")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 4.00 + 18.00 // Gemini 3.1 Pro long context pricing
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter gemini = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_Kimi(t *testing.T) {
	stats := NewSessionStats("openrouter", "moonshotai/kimi-k2.5")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.60 + 3.00 // Kimi K2.5 pricing
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter kimi k2.5 = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_KimiK2(t *testing.T) {
	stats := NewSessionStats("openrouter", "moonshotai/kimi-k2")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.60 + 2.50 // Kimi K2 pricing
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter kimi k2 = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_Llama(t *testing.T) {
	stats := NewSessionStats("openrouter", "meta/llama-3.1-70b")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.20 + 0.80 // Llama pricing
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter llama = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_GLM5(t *testing.T) {
	stats := NewSessionStats("openrouter", "zhipu/glm-5")
	stats.AddTokens(790000, 0)

	cost := stats.EstimatedCost()
	expected := 0.72 * 0.79
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter glm-5 = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenRouter_Default(t *testing.T) {
	stats := NewSessionStats("openrouter", "unknown/some-model")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.28 + 0.42 // DeepSeek V3.2 fallback
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for openrouter unknown = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_UnknownProvider(t *testing.T) {
	stats := NewSessionStats("unknown")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	// デフォルトはDeepSeek V3.2料金
	expected := 0.28 + 0.42

	// 浮動小数点の比較は許容誤差で
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for unknown provider = %f, want %f", cost, expected)
	}
}

func TestSessionStats_ElapsedTime(t *testing.T) {
	stats := NewSessionStats("test")
	stats.StartTime = time.Now().Add(-100 * time.Millisecond)

	elapsed := stats.ElapsedTime()
	if elapsed < 100*time.Millisecond {
		t.Errorf("ElapsedTime() = %v, should be at least 100ms", elapsed)
	}
}

func TestSessionStats_FormatElapsedTime_Seconds(t *testing.T) {
	stats := NewSessionStats("test")
	stats.StartTime = time.Now().Add(-30 * time.Second)

	formatted := stats.FormatElapsedTime()
	if formatted != "30s" {
		t.Errorf("FormatElapsedTime() = %q, want '30s'", formatted)
	}
}

func TestSessionStats_FormatElapsedTime_Minutes(t *testing.T) {
	stats := NewSessionStats("test")
	stats.StartTime = time.Now().Add(-2*time.Minute - 15*time.Second)

	formatted := stats.FormatElapsedTime()
	if formatted != "2m 15s" {
		t.Errorf("FormatElapsedTime() = %q, want '2m 15s'", formatted)
	}
}

func TestSessionStats_FormatElapsedTime_Hours(t *testing.T) {
	stats := NewSessionStats("test")
	stats.StartTime = time.Now().Add(-1*time.Hour - 5*time.Minute - 30*time.Second)

	formatted := stats.FormatElapsedTime()
	if formatted != "1h 5m 30s" {
		t.Errorf("FormatElapsedTime() = %q, want '1h 5m 30s'", formatted)
	}
}

func TestSessionStats_TotalMessages(t *testing.T) {
	stats := NewSessionStats("test")
	stats.UserMessages = 5
	stats.AssistantMessages = 10

	total := stats.TotalMessages()
	if total != 15 {
		t.Errorf("TotalMessages() = %d, want 15", total)
	}
}

func TestSessionStats_TotalToolExecutions(t *testing.T) {
	stats := NewSessionStats("test")
	stats.AddToolExecution("read_file")
	stats.AddToolExecution("read_file")
	stats.AddToolExecution("write_file")
	stats.AddToolExecution("bash")

	total := stats.TotalToolExecutions()
	if total != 4 {
		t.Errorf("TotalToolExecutions() = %d, want 4", total)
	}
}

func TestOptimizationMetrics_Add(t *testing.T) {
	metrics := OptimizationMetrics{
		NegativeCacheHits: 2,
		OutlineFirstCount: 1,
	}

	metrics.add(OptimizationMetrics{
		NegativeCacheHits:     4,
		CompactionCount:       1,
		CostAwareCompressions: 1,
	})
	metrics.addCompaction(CompactionMetrics{
		ErrorCompressions:      5,
		FailedPairCompressions: 6,
	})

	if metrics.NegativeCacheHits != 6 {
		t.Fatalf("NegativeCacheHits = %d, want 6", metrics.NegativeCacheHits)
	}
	if metrics.ErrorCompressions != 5 {
		t.Fatalf("ErrorCompressions = %d, want 5", metrics.ErrorCompressions)
	}
	if metrics.FailedPairCompressions != 6 {
		t.Fatalf("FailedPairCompressions = %d, want 6", metrics.FailedPairCompressions)
	}
	if metrics.OutlineFirstCount != 1 {
		t.Fatalf("OutlineFirstCount = %d, want 1", metrics.OutlineFirstCount)
	}
	if metrics.CompactionCount != 1 {
		t.Fatalf("CompactionCount = %d, want 1", metrics.CompactionCount)
	}
	if metrics.CostAwareCompressions != 1 {
		t.Fatalf("CostAwareCompressions = %d, want 1", metrics.CostAwareCompressions)
	}
}

func TestGetSessionFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "session.jsonl")

	// テストファイルを作成
	content := []byte("test session data")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := GetSessionFileSize(tmpFile)
	if err != nil {
		t.Fatalf("GetSessionFileSize() error = %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("GetSessionFileSize() = %d, want %d", size, len(content))
	}
}

func TestGetSessionFileSize_NonExistent(t *testing.T) {
	_, err := GetSessionFileSize("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("GetSessionFileSize() should return error for non-existent file")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "kilobytes",
			bytes: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "megabytes",
			bytes: 1048576, // 1MB
			want:  "1.0 MB",
		},
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{12345, "12.3k"},
		{999999, "1000.0k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}

	for _, tt := range tests {
		result := FormatTokens(tt.input)
		if result != tt.expected {
			t.Errorf("FormatTokens(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateRequestCost(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		input    int
		output   int
		expected float64
	}{
		// Ollama: 常に0
		{"ollama", "", 1000000, 1000000, 0.0},
		// DeepSeek V3.2: $0.28/1M input, $0.42/1M output
		{"deepseek", "", 1000000, 1000000, 0.70},
		// Unknown: DeepSeek V3.2料金と同じ
		{"unknown", "", 1000000, 1000000, 0.70},
		// Claude Sonnet（1M input > 200K → long context 料金）
		{"claude", "claude-sonnet-4-5-20250929", 1000000, 1000000, 6.00 + 22.50},
		// Claude Opus（1M input > 200K → long context 料金）
		{"claude", "claude-opus-4-5", 1000000, 1000000, 10.00 + 37.50},
		// Claude Haiku（long context なし）
		{"claude", "claude-haiku-4-5", 1000000, 1000000, 1.00 + 5.00},
		// Claude Sonnet 通常料金（<=200K）
		{"claude", "claude-sonnet-4-5-20250929", 100000, 100000, 0.30 + 1.50},
		// Claude Opus 通常料金（<=200K）
		{"claude", "claude-opus-4-5", 100000, 100000, 0.50 + 2.50},
	}

	for _, tt := range tests {
		result := CalculateRequestCost(tt.provider, tt.model, tt.input, tt.output)
		// 浮動小数点の比較は許容誤差を設ける
		if result < tt.expected-0.001 || result > tt.expected+0.001 {
			t.Errorf("CalculateRequestCost(%s, %s, %d, %d) = %f, want %f",
				tt.provider, tt.model, tt.input, tt.output, result, tt.expected)
		}
	}
}

func TestGetGeminiPricing_200KTier(t *testing.T) {
	// Pro: 200K以下 → 標準料金
	p1 := getGeminiPricing("gemini-3.1-pro", 100000)
	if p1.InputCostPerM != 2.00 {
		t.Errorf("Gemini Pro <200K input = %f, want 2.00", p1.InputCostPerM)
	}
	if p1.OutputCostPerM != 12.00 {
		t.Errorf("Gemini Pro <200K output = %f, want 12.00", p1.OutputCostPerM)
	}

	// Pro: 200K超 → 高ティア料金
	p2 := getGeminiPricing("gemini-3.1-pro", 250000)
	if p2.InputCostPerM != 4.00 {
		t.Errorf("Gemini Pro >200K input = %f, want 4.00", p2.InputCostPerM)
	}
	if p2.OutputCostPerM != 18.00 {
		t.Errorf("Gemini Pro >200K output = %f, want 18.00", p2.OutputCostPerM)
	}
	if p2.CachedInputCostPerM != 0.40 {
		t.Errorf("Gemini Pro >200K cached = %f, want 0.40", p2.CachedInputCostPerM)
	}

	// Flash: 200K超でも料金変わらない（long context ティアなし）
	f1 := getGeminiPricing("gemini-3-flash", 250000)
	if f1.InputCostPerM != 0.50 {
		t.Errorf("Gemini Flash >200K input = %f, want 0.50 (no long context tier)", f1.InputCostPerM)
	}
}

func TestEstimatedCost_GeminiTierTransition(t *testing.T) {
	stats := NewSessionStats("gemini", "gemini-3.1-pro")

	// Request 1: 100K input, 10K output (標準ティア)
	stats.AddUsage(api.Usage{
		InputTokens:  100000,
		OutputTokens: 10000,
	})
	cost1 := stats.EstimatedCost()
	// 100K/1M * $2.00 + 10K/1M * $12.00 = $0.20 + $0.12 = $0.32
	expected1 := 0.32
	if cost1 < expected1-0.001 || cost1 > expected1+0.001 {
		t.Errorf("After req1: cost = %f, want %f", cost1, expected1)
	}

	// Request 2: 250K input, 20K output (高ティア)
	stats.AddUsage(api.Usage{
		InputTokens:  250000,
		OutputTokens: 20000,
	})
	cost2 := stats.EstimatedCost()
	// req2: 250K/1M * $4.00 + 20K/1M * $18.00 = $1.00 + $0.36 = $1.36
	// 合計: $0.32 + $1.36 = $1.68
	expected2 := 1.68
	if cost2 < expected2-0.001 || cost2 > expected2+0.001 {
		t.Errorf("After req2: cost = %f, want %f", cost2, expected2)
	}
}

func TestCalculateRequestCostWithCache_GeminiTier(t *testing.T) {
	// 200K以下
	cost1 := CalculateRequestCostWithCache("gemini", "gemini-3.1-pro", api.Usage{
		InputTokens:       100000,
		OutputTokens:      10000,
		CachedInputTokens: 50000,
	})
	// uncached: 50K/1M * $2.00 = $0.10
	// cached:   50K/1M * $0.20 = $0.01
	// output:   10K/1M * $12.00 = $0.12
	// total: $0.23
	expected1 := 0.23
	if cost1 < expected1-0.001 || cost1 > expected1+0.001 {
		t.Errorf("CostWithCache <200K = %f, want %f", cost1, expected1)
	}

	// 200K超
	cost2 := CalculateRequestCostWithCache("gemini", "gemini-3.1-pro", api.Usage{
		InputTokens:       300000,
		OutputTokens:      10000,
		CachedInputTokens: 100000,
	})
	// uncached: 200K/1M * $4.00 = $0.80
	// cached:   100K/1M * $0.40 = $0.04
	// output:   10K/1M * $18.00 = $0.18
	// total: $1.02
	expected2 := 1.02
	if cost2 < expected2-0.001 || cost2 > expected2+0.001 {
		t.Errorf("CostWithCache >200K = %f, want %f", cost2, expected2)
	}
}

func TestSessionStats_AddUsage_ThinkingTokens(t *testing.T) {
	stats := NewSessionStats("gemini", "gemini-3.1-pro")

	usage := api.Usage{
		InputTokens:    100000,
		OutputTokens:   10000,
		ThinkingTokens: 50000,
	}
	stats.AddUsage(usage)

	if stats.ThinkingTokens != 50000 {
		t.Errorf("AddUsage() ThinkingTokens = %d, want 50000", stats.ThinkingTokens)
	}

	// TotalTokens = Input + Output + Thinking
	if stats.TotalTokens() != 160000 {
		t.Errorf("TotalTokens() = %d, want 160000", stats.TotalTokens())
	}

	// コスト検証: thinking tokens は出力レートで課金
	// Input: 100K/1M * $2.00 = $0.20
	// Output: 10K/1M * $12.00 = $0.12
	// Thinking: 50K/1M * $12.00 = $0.60
	// 合計: $0.92
	expected := 0.92
	cost := stats.EstimatedCost()
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() with thinking = %f, want %f", cost, expected)
	}
}

func TestCalculateRequestCostWithCache_ThinkingTokens(t *testing.T) {
	cost := CalculateRequestCostWithCache("gemini", "gemini-3.1-pro", api.Usage{
		InputTokens:    100000,
		OutputTokens:   10000,
		ThinkingTokens: 50000,
	})
	// Input: 100K/1M * $2.00 = $0.20
	// Output: 10K/1M * $12.00 = $0.12
	// Thinking: 50K/1M * $12.00 = $0.60
	// 合計: $0.92
	expected := 0.92
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("CostWithCache thinking = %f, want %f", cost, expected)
	}
}

func TestGetClaudePricing(t *testing.T) {
	tests := []struct {
		model      string
		ptc        int // promptTokenCount
		wantInput  float64
		wantOutput float64
	}{
		// 通常料金 (<=200K)
		{"claude-opus-4-5-20251101", 0, 5.00, 25.00},
		{"claude-opus-4-6", 0, 5.00, 25.00},
		{"global.anthropic.claude-opus-4-5-20251101-v1:0", 0, 5.00, 25.00},
		{"claude-haiku-4-5-20251001", 0, 1.00, 5.00},
		{"claude-sonnet-4-5-20250929", 0, 3.00, 15.00},
		{"claude-sonnet-4-20250514", 0, 3.00, 15.00},
		{"", 0, 3.00, 15.00},                   // 空文字列はSonnetデフォルト
		{"some-unknown-model", 0, 3.00, 15.00}, // 不明モデルもSonnetデフォルト

		// Long context (>200K)
		{"claude-opus-4-6", 250000, 10.00, 37.50},
		{"claude-sonnet-4-5-20250929", 250000, 6.00, 22.50},
		{"", 250000, 6.00, 22.50}, // Sonnet デフォルト long context

		// Haiku は long context なし
		{"claude-haiku-4-5-20251001", 250000, 1.00, 5.00},
	}

	for _, tt := range tests {
		pricing := getClaudePricing(tt.model, tt.ptc)
		if pricing.InputCostPerM != tt.wantInput {
			t.Errorf("getClaudePricing(%q, %d).InputCostPerM = %f, want %f", tt.model, tt.ptc, pricing.InputCostPerM, tt.wantInput)
		}
		if pricing.OutputCostPerM != tt.wantOutput {
			t.Errorf("getClaudePricing(%q, %d).OutputCostPerM = %f, want %f", tt.model, tt.ptc, pricing.OutputCostPerM, tt.wantOutput)
		}
	}
}

func TestGetPricingInfo_FallbackToHardcodedWhenYAMLUnavailable(t *testing.T) {
	origEmbedded := embeddedPricingYAML
	origCfg := loadedPricingConfig

	embeddedPricingYAML = nil
	pricingConfigOnce = sync.Once{}
	loadedPricingConfig = nil
	t.Cleanup(func() {
		embeddedPricingYAML = origEmbedded
		pricingConfigOnce = sync.Once{}
		loadedPricingConfig = origCfg
	})

	pricing := GetPricingInfo("openai", "gpt-5")
	if pricing.InputCostPerM != 1.25 || pricing.OutputCostPerM != 10.00 {
		t.Fatalf("fallback pricing mismatch: got input=%f output=%f", pricing.InputCostPerM, pricing.OutputCostPerM)
	}
}

func TestGetOpenAIPricing_GPT54(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4")
	if pricing.InputCostPerM != 2.50 {
		t.Errorf("expected 2.50, got %f", pricing.InputCostPerM)
	}
	if pricing.OutputCostPerM != 15.00 {
		t.Errorf("expected 15.00, got %f", pricing.OutputCostPerM)
	}
}

func TestGetOpenAIPricing_GPT54Pro(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4-pro")
	if pricing.InputCostPerM != 30.00 {
		t.Errorf("expected 30.00, got %f", pricing.InputCostPerM)
	}
	if pricing.OutputCostPerM != 180.00 {
		t.Errorf("expected 180.00, got %f", pricing.OutputCostPerM)
	}
}

func TestGetOpenAIPricing_GPT54_LongInput(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4", 300000)
	if pricing.InputCostPerM != 5.00 {
		t.Errorf("expected 5.00 for long input, got %f", pricing.InputCostPerM)
	}
	if pricing.CachedInputCostPerM != 0.50 {
		t.Errorf("expected 0.50 cached long input, got %f", pricing.CachedInputCostPerM)
	}
}

func TestGetOpenAIPricing_GPT54Pro_LongInput(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4-pro", 300000)
	if pricing.InputCostPerM != 60.00 {
		t.Errorf("expected 60.00 for long input, got %f", pricing.InputCostPerM)
	}
	if pricing.CachedInputCostPerM != 6.00 {
		t.Errorf("expected 6.00 cached long input, got %f", pricing.CachedInputCostPerM)
	}
}

// === 新規テスト: DeepSeek V3.2 統一料金 ===

func TestGetDeepSeekPricing_V32Unified(t *testing.T) {
	// deepseek-chat と deepseek-reasoner は V3.2 で統一料金
	tests := []struct {
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{"deepseek-chat", 0.28, 0.42},
		{"deepseek-reasoner", 0.28, 0.42}, // V3.2 で統一
		{"", 0.28, 0.42},                  // デフォルト
	}
	for _, tt := range tests {
		pricing := getDeepSeekPricing(tt.model)
		if pricing.InputCostPerM != tt.wantInput {
			t.Errorf("getDeepSeekPricing(%q).InputCostPerM = %f, want %f", tt.model, pricing.InputCostPerM, tt.wantInput)
		}
		if pricing.OutputCostPerM != tt.wantOutput {
			t.Errorf("getDeepSeekPricing(%q).OutputCostPerM = %f, want %f", tt.model, pricing.OutputCostPerM, tt.wantOutput)
		}
	}
}

// === 新規テスト: Claude long context 200K超 ===

func TestGetClaudePricing_LongContext(t *testing.T) {
	// Sonnet: >200K → $6/$22.50
	p := getClaudePricing("claude-sonnet-4-5-20250929", 250000)
	if p.InputCostPerM != 6.00 || p.OutputCostPerM != 22.50 {
		t.Errorf("Sonnet long context: got input=%f output=%f, want 6.00/22.50", p.InputCostPerM, p.OutputCostPerM)
	}
	if p.CachedInputCostPerM != 0.60 {
		t.Errorf("Sonnet long context cached: got %f, want 0.60", p.CachedInputCostPerM)
	}

	// Opus: >200K → $10/$37.50
	p2 := getClaudePricing("claude-opus-4-6", 250000)
	if p2.InputCostPerM != 10.00 || p2.OutputCostPerM != 37.50 {
		t.Errorf("Opus long context: got input=%f output=%f, want 10.00/37.50", p2.InputCostPerM, p2.OutputCostPerM)
	}

	// Haiku: long context なし
	p3 := getClaudePricing("claude-haiku-4-5", 250000)
	if p3.InputCostPerM != 1.00 || p3.OutputCostPerM != 5.00 {
		t.Errorf("Haiku should not have long context pricing: got input=%f output=%f", p3.InputCostPerM, p3.OutputCostPerM)
	}
}

// === 新規テスト: 200Kティア判定がキャッシュトークンを含む ===

func TestCalculateRequestCostWithCache_TierIncludesCachedTokens(t *testing.T) {
	// Claude Sonnet: 通常input 150K + cached 60K = 210K → long context 料金
	cost := CalculateRequestCostWithCache("claude", "claude-sonnet-4-5", api.Usage{
		InputTokens:       150000,
		OutputTokens:      10000,
		CachedInputTokens: 60000,
	})
	// totalInputForTier = 150K + 60K + 0 = 210K → long context ($6/$22.50)
	// uncached: (150K-60K)/1M * $6.00 = 90K * 0.000006 = $0.54
	// cached:   60K/1M * $0.60 = $0.036
	// output:   10K/1M * $22.50 = $0.225
	// total: $0.801
	expected := 0.801
	if cost < expected-0.01 || cost > expected+0.01 {
		t.Errorf("CostWithCache (cached triggers 200K tier) = %f, want ~%f", cost, expected)
	}

	// 同じトークン数でもキャッシュなし 150K → 通常料金
	cost2 := CalculateRequestCostWithCache("claude", "claude-sonnet-4-5", api.Usage{
		InputTokens:  150000,
		OutputTokens: 10000,
	})
	// totalInputForTier = 150K → 通常 ($3/$15)
	// uncached: 150K/1M * $3.00 = $0.45
	// output:   10K/1M * $15.00 = $0.15
	// total: $0.60
	expected2 := 0.60
	if cost2 < expected2-0.01 || cost2 > expected2+0.01 {
		t.Errorf("CostWithCache (no cache, <200K) = %f, want ~%f", cost2, expected2)
	}
}

// === 新規テスト: Gemini 2.5 Pro ===

func TestGetGeminiPricing_25Pro(t *testing.T) {
	// 2.5 Pro: $1.25/$10.00 (<=200K)
	p := getGeminiPricing("gemini-2.5-pro", 100000)
	if p.InputCostPerM != 1.25 || p.OutputCostPerM != 10.00 {
		t.Errorf("Gemini 2.5 Pro <200K: got input=%f output=%f, want 1.25/10.00", p.InputCostPerM, p.OutputCostPerM)
	}

	// 2.5 Pro: $2.50/$15.00 (>200K)
	p2 := getGeminiPricing("gemini-2.5-pro", 250000)
	if p2.InputCostPerM != 2.50 || p2.OutputCostPerM != 15.00 {
		t.Errorf("Gemini 2.5 Pro >200K: got input=%f output=%f, want 2.50/15.00", p2.InputCostPerM, p2.OutputCostPerM)
	}
}

// === 新規テスト: Gemini Flash-Lite 分離 ===

func TestGetGeminiPricing_FlashLite(t *testing.T) {
	// 2.5 Flash-Lite: $0.10/$0.40
	p := getGeminiPricing("gemini-2.5-flash-lite", 0)
	if p.InputCostPerM != 0.10 || p.OutputCostPerM != 0.40 {
		t.Errorf("Gemini 2.5 Flash-Lite: got input=%f output=%f, want 0.10/0.40", p.InputCostPerM, p.OutputCostPerM)
	}

	// 2.5 Flash: $0.30/$2.50 (Flash-Lite とは違う)
	p2 := getGeminiPricing("gemini-2.5-flash", 0)
	if p2.InputCostPerM != 0.30 || p2.OutputCostPerM != 2.50 {
		t.Errorf("Gemini 2.5 Flash: got input=%f output=%f, want 0.30/2.50", p2.InputCostPerM, p2.OutputCostPerM)
	}
}

// === 新規テスト: Gemini 2.0 Flash ===

func TestGetGeminiPricing_20Flash(t *testing.T) {
	p := getGeminiPricing("gemini-2.0-flash", 0)
	if p.InputCostPerM != 0.10 || p.OutputCostPerM != 0.40 {
		t.Errorf("Gemini 2.0 Flash: got input=%f output=%f, want 0.10/0.40", p.InputCostPerM, p.OutputCostPerM)
	}

	// 2.0 Flash-Lite
	p2 := getGeminiPricing("gemini-2.0-flash-lite", 0)
	if p2.InputCostPerM != 0.075 || p2.OutputCostPerM != 0.30 {
		t.Errorf("Gemini 2.0 Flash-Lite: got input=%f output=%f, want 0.075/0.30", p2.InputCostPerM, p2.OutputCostPerM)
	}
}

// === 新規テスト: GPT-5.4-mini / GPT-5.4-nano 専用価格 ===

func TestGetOpenAIPricing_GPT54Mini(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4-mini")
	if pricing.InputCostPerM != 0.75 {
		t.Errorf("gpt-5.4-mini input: got %f, want 0.75", pricing.InputCostPerM)
	}
	if pricing.CachedInputCostPerM != 0.075 {
		t.Errorf("gpt-5.4-mini cached input: got %f, want 0.075", pricing.CachedInputCostPerM)
	}
	if pricing.OutputCostPerM != 4.50 {
		t.Errorf("gpt-5.4-mini output: got %f, want 4.50", pricing.OutputCostPerM)
	}
}

func TestGetOpenAIPricing_GPT54Nano(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-5.4-nano")
	if pricing.InputCostPerM != 0.20 {
		t.Errorf("gpt-5.4-nano input: got %f, want 0.20", pricing.InputCostPerM)
	}
	if pricing.CachedInputCostPerM != 0.02 {
		t.Errorf("gpt-5.4-nano cached input: got %f, want 0.02", pricing.CachedInputCostPerM)
	}
	if pricing.OutputCostPerM != 1.25 {
		t.Errorf("gpt-5.4-nano output: got %f, want 1.25", pricing.OutputCostPerM)
	}
}

// === 新規テスト: Gemini 3.1 Flash-Lite Preview 専用価格 ===

func TestGetGeminiPricing_31FlashLite(t *testing.T) {
	pricing := getGeminiPricing("gemini-3.1-flash-lite-preview", 0)
	if pricing.InputCostPerM != 0.25 {
		t.Errorf("gemini-3.1-flash-lite-preview input: got %f, want 0.25", pricing.InputCostPerM)
	}
	if pricing.CachedInputCostPerM != 0.025 {
		t.Errorf("gemini-3.1-flash-lite-preview cached input: got %f, want 0.025", pricing.CachedInputCostPerM)
	}
	if pricing.OutputCostPerM != 1.50 {
		t.Errorf("gemini-3.1-flash-lite-preview output: got %f, want 1.50", pricing.OutputCostPerM)
	}
}

// === 新規テスト: 汎用 mini/nano が 5.4 以外では従来通り ===

func TestGetOpenAIPricing_GenericMiniStillWorks(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-4.1-mini")
	if pricing.InputCostPerM != 0.25 {
		t.Errorf("gpt-4.1-mini input: got %f, want 0.25 (generic mini)", pricing.InputCostPerM)
	}
}

func TestGetOpenAIPricing_GenericNanoStillWorks(t *testing.T) {
	pricing := GetPricingInfo("openai", "gpt-4.1-nano")
	if pricing.InputCostPerM != 0.05 {
		t.Errorf("gpt-4.1-nano input: got %f, want 0.05 (generic nano)", pricing.InputCostPerM)
	}
}

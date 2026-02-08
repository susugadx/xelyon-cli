package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	expected := 0.14 + 0.28 // $0.14/1M + $0.28/1M

	// 浮動小数点の比較は許容誤差で
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for deepseek = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_OpenAI(t *testing.T) {
	stats := NewSessionStats("openai")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 2.50 + 10.00
	if cost != expected {
		t.Errorf("EstimatedCost() for openai = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude(t *testing.T) {
	// デフォルト（model=""）は Sonnet 料金
	stats := NewSessionStats("claude")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 3.00 + 15.00
	if cost != expected {
		t.Errorf("EstimatedCost() for claude (sonnet default) = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude_Opus(t *testing.T) {
	stats := NewSessionStats("claude", "claude-opus-4-5-20251101")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 5.00 + 25.00
	if cost != expected {
		t.Errorf("EstimatedCost() for claude opus = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Claude_Haiku(t *testing.T) {
	stats := NewSessionStats("claude", "claude-haiku-4-5-20251001")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.80 + 4.00
	if cost != expected {
		t.Errorf("EstimatedCost() for claude haiku = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Bedrock_Opus(t *testing.T) {
	stats := NewSessionStats("bedrock", "global.anthropic.claude-opus-4-5-20251101-v1:0")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 5.00 + 25.00
	if cost != expected {
		t.Errorf("EstimatedCost() for bedrock opus = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Gemini(t *testing.T) {
	stats := NewSessionStats("gemini")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.075 + 0.30
	if cost != expected {
		t.Errorf("EstimatedCost() for gemini = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_Groq(t *testing.T) {
	stats := NewSessionStats("groq")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	expected := 0.10 + 0.10
	if cost != expected {
		t.Errorf("EstimatedCost() for groq = %f, want %f", cost, expected)
	}
}

func TestSessionStats_EstimatedCost_UnknownProvider(t *testing.T) {
	stats := NewSessionStats("unknown")
	stats.AddTokens(1000000, 1000000)

	cost := stats.EstimatedCost()
	// デフォルトはDeepSeek料金
	expected := 0.14 + 0.28

	// 浮動小数点の比較は許容誤差で
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("EstimatedCost() for unknown provider = %f, want %f", cost, expected)
	}
}

func TestSessionStats_ElapsedTime(t *testing.T) {
	stats := NewSessionStats("test")
	time.Sleep(100 * time.Millisecond)

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
		// DeepSeek: 0.14/1M input, 0.28/1M output
		{"deepseek", "", 1000000, 1000000, 0.42},
		// Unknown: DeepSeek料金と同じ
		{"unknown", "", 1000000, 1000000, 0.42},
		// Claude Sonnet（デフォルト）
		{"claude", "claude-sonnet-4-5-20250929", 1000000, 1000000, 3.00 + 15.00},
		// Claude Opus
		{"claude", "claude-opus-4-5", 1000000, 1000000, 5.00 + 25.00},
		// Claude Haiku
		{"claude", "claude-haiku-4-5", 1000000, 1000000, 0.80 + 4.00},
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

func TestGetClaudePricing(t *testing.T) {
	tests := []struct {
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{"claude-opus-4-5-20251101", 5.00, 25.00},
		{"claude-opus-4-6", 5.00, 25.00},
		{"global.anthropic.claude-opus-4-5-20251101-v1:0", 5.00, 25.00},
		{"claude-haiku-4-5-20251001", 0.80, 4.00},
		{"claude-sonnet-4-5-20250929", 3.00, 15.00},
		{"claude-sonnet-4-20250514", 3.00, 15.00},
		{"", 3.00, 15.00},                   // 空文字列はSonnetデフォルト
		{"some-unknown-model", 3.00, 15.00}, // 不明モデルもSonnetデフォルト
	}

	for _, tt := range tests {
		pricing := getClaudePricing(tt.model)
		if pricing.InputCostPerM != tt.wantInput {
			t.Errorf("getClaudePricing(%q).InputCostPerM = %f, want %f", tt.model, pricing.InputCostPerM, tt.wantInput)
		}
		if pricing.OutputCostPerM != tt.wantOutput {
			t.Errorf("getClaudePricing(%q).OutputCostPerM = %f, want %f", tt.model, pricing.OutputCostPerM, tt.wantOutput)
		}
	}
}

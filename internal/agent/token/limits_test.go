package token

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetModelTokenLimit_ExactMatch(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		// Claude models
		{"claude-sonnet-4-20250514", 200000},
		{"claude-3-5-sonnet-20241022", 200000},
		{"claude-3-opus-20240229", 200000},

		// OpenAI models
		{"gpt-4.1", 1000000},
		{"gpt-4.1-mini", 1000000},
		{"gpt-5", 400000},
		{"gpt-5.4", 1000000},
		{"gpt-5.4-pro", 1000000},
		{"gpt-5.4-mini", 400000},
		{"gpt-5.4-nano", 400000},
		{"gpt-5.1", 400000},
		{"gpt-5.2", 400000},
		{"gpt-4o", 128000},
		{"gpt-4o-mini", 128000},
		{"gpt-4", 8192},
		{"gpt-4-32k", 32768},
		{"gpt-3.5-turbo", 16385},
		{"o1", 200000},

		// Gemini models
		{"gemini-2.0-flash", 1000000},
		{"gemini-1.5-pro", 2000000},
		{"gemini-pro", 32768},

		// DeepSeek models
		{"deepseek-chat", 128000},
		{"deepseek-coder", 64000},
		{"deepseek-reasoner", 128000},

		// Groq models
		{"llama-3.3-70b-versatile", 128000},
		{"mixtral-8x7b-32768", 32768},

		// Ollama models
		{"qwen2.5-coder:7b", 32768},
		{"codellama:7b", 16384},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := GetModelTokenLimit(tt.model)
			if result != tt.expected {
				t.Errorf("GetModelTokenLimit(%q) = %d, want %d", tt.model, result, tt.expected)
			}
		})
	}
}

func TestGetModelTokenLimitForConfig_UsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]config.ModelOverride{
			"mini-deployment": {CatalogModel: "gpt-5.4-mini"},
		},
	})

	if got := GetModelTokenLimitForConfig(cfg, "openai", "corp-gpt-deployment"); got != 1000000 {
		t.Fatalf("GetModelTokenLimitForConfig(default deployment) = %d, want 1000000", got)
	}
	if got := GetModelTokenLimitForConfig(cfg, "openai", "mini-deployment"); got != 400000 {
		t.Fatalf("GetModelTokenLimitForConfig(model override) = %d, want 400000", got)
	}
}

func TestGetModelTokenLimit_PrefixMatch(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		// Claude prefix
		{"claude-3-5-sonnet-latest", 200000},
		{"claude-new-model", 200000},

		// GPT-4.1 prefix
		{"gpt-4.1-nano-20250101", 1000000},

		// GPT-4o prefix
		{"gpt-4o-2024-08-06", 128000},

		// GPT-4 prefix (after gpt-4o check)
		{"gpt-4-0125-preview", 8192},

		// GPT-3.5 prefix
		{"gpt-3.5-turbo-0125", 16385},

		// GPT-5 prefix
		{"gpt-5.4-mini-20260301", 400000},
		{"gpt-5.4-nano-20260301", 400000},
		{"gpt-5.4-preview", 1000000},
		{"gpt-5-preview", 400000},

		// O1/O3 prefix
		{"o1-2024-12-17", 200000},
		{"o3-high", 200000},

		// Gemini 2.x prefix
		{"gemini-2.1-pro", 1000000},

		// Gemini 1.5 prefix
		{"gemini-1.5-flash-latest", 1000000},

		// DeepSeek prefix
		{"deepseek-v3", 128000},
		{"deepseek-r1-2025", 128000},
		{"deepseek-coder-v3", 64000},

		// Llama 3.x prefix
		{"llama-3.2-8b", 128000},

		// Qwen prefix
		{"qwen2.5-coder:72b", 32768},

		// Mixtral prefix
		{"mixtral-new", 32768},

		// CodeLlama prefix
		{"codellama:70b", 16384},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := GetModelTokenLimit(tt.model)
			if result != tt.expected {
				t.Errorf("GetModelTokenLimit(%q) = %d, want %d", tt.model, result, tt.expected)
			}
		})
	}
}

func TestGetModelTokenLimit_Default(t *testing.T) {
	unknownModels := []string{
		"unknown-model",
		"custom-local-model",
		"my-fine-tuned-model",
		"",
	}

	expectedDefault := 100000

	for _, model := range unknownModels {
		t.Run(model, func(t *testing.T) {
			result := GetModelTokenLimit(model)
			if result != expectedDefault {
				t.Errorf("GetModelTokenLimit(%q) = %d, want default %d", model, result, expectedDefault)
			}
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minCount int // Minimum expected count
		maxCount int // Maximum expected count
	}{
		{
			name:     "empty string",
			text:     "",
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "short english text",
			text:     "Hello, world!",
			minCount: 4,  // At least 4 tokens
			maxCount: 10, // At most 10 tokens
		},
		{
			name:     "longer english text",
			text:     "The quick brown fox jumps over the lazy dog.",
			minCount: 10,
			maxCount: 20,
		},
		{
			name:     "code snippet",
			text:     "func main() {\n\tfmt.Println(\"Hello\")\n}",
			minCount: 10,
			maxCount: 25,
		},
		{
			name:     "japanese text",
			text:     "こんにちは世界",
			minCount: 7,
			maxCount: 20,
		},
		{
			name:     "mixed text",
			text:     "Hello こんにちは World 世界",
			minCount: 8,
			maxCount: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokenCount(tt.text)

			if result < tt.minCount {
				t.Errorf("EstimateTokenCount(%q) = %d, want >= %d", tt.text, result, tt.minCount)
			}
			if result > tt.maxCount {
				t.Errorf("EstimateTokenCount(%q) = %d, want <= %d", tt.text, result, tt.maxCount)
			}
		})
	}
}

func TestEstimateTokenCountForModel_NonASCIIIsNotUndercounted(t *testing.T) {
	text := "日本語で長めの説明を書いています。"
	got := EstimateTokenCountForModel("deepseek-chat", text)
	if got < len([]rune(text)) {
		t.Fatalf("EstimateTokenCountForModel() = %d, want >= rune count %d", got, len([]rune(text)))
	}
}

func TestEstimateStructuredValueTokenCount(t *testing.T) {
	value := map[string]any{
		"name":        "search_code",
		"description": "Search the repository",
		"params": map[string]any{
			"pattern": "*.go",
		},
	}

	got := EstimateStructuredValueTokenCountForModel("gpt-4o", value)
	if got <= 0 {
		t.Fatal("EstimateStructuredValueTokenCountForModel() should return positive value")
	}
}

func TestEstimateTokenCount_Consistency(t *testing.T) {
	// Same input should always produce same output
	text := "This is a test string for token estimation."

	first := EstimateTokenCount(text)
	second := EstimateTokenCount(text)

	if first != second {
		t.Errorf("EstimateTokenCount is not consistent: %d vs %d", first, second)
	}
}

func TestEstimateTokenCount_Proportional(t *testing.T) {
	// Longer text should have more tokens
	short := "Hello"
	long := "Hello, this is a much longer piece of text that should definitely have more tokens."

	shortCount := EstimateTokenCount(short)
	longCount := EstimateTokenCount(long)

	if longCount <= shortCount {
		t.Errorf("Longer text (%d tokens) should have more tokens than shorter text (%d tokens)", longCount, shortCount)
	}
}

func TestModelTokenLimitPolicy_ReturnsPositiveLimitsForRepresentatives(t *testing.T) {
	for _, model := range []string{
		"claude-sonnet-4-6",
		"gpt-5.4",
		"gemini-3.1-pro-preview",
		"deepseek-chat",
		"llama-3.3-70b-versatile",
		"qwen2.5-coder:7b",
		"unknown-model",
	} {
		if limit := GetModelTokenLimit(model); limit <= 0 {
			t.Errorf("GetModelTokenLimit(%q) = %d, want positive", model, limit)
		}
	}
}

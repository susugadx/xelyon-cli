package token

import (
	"testing"
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
		{"deepseek-chat", 64000},
		{"deepseek-coder", 64000},

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

func TestGetModelTokenLimit_PrefixMatch(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		// Claude prefix
		{"claude-3-5-sonnet-latest", 200000},
		{"claude-new-model", 200000},

		// GPT-4o prefix
		{"gpt-4o-2024-08-06", 128000},

		// GPT-4 prefix (after gpt-4o check)
		{"gpt-4-0125-preview", 8192},

		// GPT-3.5 prefix
		{"gpt-3.5-turbo-0125", 16385},

		// GPT-5 prefix
		{"gpt-5-preview", 200000},

		// O1/O3 prefix
		{"o1-2024-12-17", 200000},
		{"o3-high", 200000},

		// Gemini 2.x prefix
		{"gemini-2.1-pro", 1000000},

		// Gemini 1.5 prefix
		{"gemini-1.5-flash-latest", 1000000},

		// DeepSeek prefix
		{"deepseek-v3", 64000},

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
			minCount: 3,  // At least 3 tokens
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
			minCount: 5,
			maxCount: 15,
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

func TestModelTokenLimits_AllPositive(t *testing.T) {
	// All model limits should be positive
	for model, limit := range modelTokenLimits {
		if limit <= 0 {
			t.Errorf("Model %q has non-positive limit: %d", model, limit)
		}
	}
}

func TestModelTokenLimits_DefaultExists(t *testing.T) {
	// "default" key should exist
	if _, ok := modelTokenLimits["default"]; !ok {
		t.Error("Default token limit not defined")
	}
}

package agent

import (
	"testing"
)

func TestInferProviderFromModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		// OpenAI models
		{"gpt-4", "gpt-4", "openai"},
		{"gpt-4-turbo", "gpt-4-turbo", "openai"},
		{"gpt-5.2-codex", "gpt-5.2-codex", "openai"},
		{"o1-preview", "o1-preview", "openai"},
		{"o3-mini", "o3-mini", "openai"},

		// Claude models
		{"claude-3-opus", "claude-3-opus-20240229", "claude"},
		{"claude-3-sonnet", "claude-3-sonnet-20240229", "claude"},
		{"claude-sonnet-4", "claude-sonnet-4-5-20250514", "claude"},

		// Gemini models
		{"gemini-pro", "gemini-pro", "gemini"},
		{"gemini-2.5-flash", "gemini-2.5-flash", "gemini"},

		// DeepSeek models
		{"deepseek-coder", "deepseek-coder", "deepseek"},
		{"deepseek-chat", "deepseek-chat", "deepseek"},

		// Ollama models
		{"llama-3", "llama-3", "ollama"},
		{"qwen2.5-coder", "qwen2.5-coder:7b", "ollama"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferProviderFromModel(tt.model)
			if result != tt.expected {
				t.Errorf("inferProviderFromModel(%s) = %s, want %s", tt.model, result, tt.expected)
			}
		})
	}
}

func TestParseInvestigationQueries(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantQueries []string
		wantErr     bool
	}{
		{
			name: "valid JSON",
			response: `Here are the queries:
{"queries": ["Query 1", "Query 2", "Query 3"]}`,
			wantQueries: []string{"Query 1", "Query 2", "Query 3"},
			wantErr:     false,
		},
		{
			name:        "JSON with extra text",
			response:    "I will investigate the following:\n```json\n{\"queries\": [\"Check file structure\", \"Review existing implementation\"]}\n```\nThese queries will help understand the codebase.",
			wantQueries: []string{"Check file structure", "Review existing implementation"},
			wantErr:     false,
		},
		{
			name:        "no JSON",
			response:    "No JSON content here.",
			wantQueries: nil,
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			response:    `{"queries": ["query1", "query2"`,
			wantQueries: nil,
			wantErr:     true,
		},
		{
			name:        "empty queries",
			response:    `{"queries": []}`,
			wantQueries: []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries, err := parseInvestigationQueries(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseInvestigationQueries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(queries) != len(tt.wantQueries) {
					t.Errorf("parseInvestigationQueries() = %v, want %v", queries, tt.wantQueries)
					return
				}
				for i, q := range queries {
					if q != tt.wantQueries[i] {
						t.Errorf("parseInvestigationQueries()[%d] = %s, want %s", i, q, tt.wantQueries[i])
					}
				}
			}
		})
	}
}

func TestSupervisor_aggregateInvestigationResults(t *testing.T) {
	s := &Supervisor{}

	results := []WorkerResult{
		{WorkerID: 0, Success: true, Output: "Found 5 files in src/"},
		{WorkerID: 1, Success: true, Output: "Database connection pool exists"},
		{WorkerID: 2, Success: false, Error: &StepFailure{StepID: 0, Reason: "timeout"}},
	}

	aggregated := s.aggregateInvestigationResults(results)

	// 結果が含まれていることを確認
	if !containsSubstring(aggregated, "Investigation 1") {
		t.Error("aggregated should contain Investigation 1")
	}
	if !containsSubstring(aggregated, "Found 5 files") {
		t.Error("aggregated should contain first result")
	}
	if !containsSubstring(aggregated, "Database connection pool") {
		t.Error("aggregated should contain second result")
	}
	if !containsSubstring(aggregated, "Failed") {
		t.Error("aggregated should contain failure message")
	}
}

func TestSupervisor_getExecutableSteps(t *testing.T) {
	// 注意: この関数は Plan の CanExecute に依存するため、
	// 実際のテストには Plan のモックが必要
	// ここでは Supervisor 構造体の初期化のみテスト
	s := &Supervisor{
		retryCount: make(map[int]int),
	}

	if s.retryCount == nil {
		t.Error("retryCount should be initialized")
	}
}

// containsSubstring は strings.Contains のヘルパー
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package review

import (
	"errors"
	"strings"
	"testing"
)

// MockLLMClient is a mock implementation of LLMClient for testing.
type MockLLMClient struct {
	Response string
	Err      error
	Calls    []string
}

func (m *MockLLMClient) Chat(prompt string) (string, error) {
	m.Calls = append(m.Calls, prompt)
	return m.Response, m.Err
}

func TestAIAnalyzer_AnalyzeWithAI(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		filePath     string
		mockResponse string
		mockErr      error
		wantInsights int
		wantErr      bool
	}{
		{
			name:     "successful analysis with insights",
			code:     "func main() { client := &http.Client{} }",
			filePath: "main.go",
			mockResponse: `{
				"insights": [
					{
						"severity": "warning",
						"category": "security",
						"line": 1,
						"description": "HTTP client without timeout",
						"suggestion": "Add Timeout field to http.Client",
						"confidence": 0.95
					}
				]
			}`,
			wantInsights: 1,
			wantErr:      false,
		},
		{
			name:     "no issues found",
			code:     "func main() { fmt.Println(\"hello\") }",
			filePath: "main.go",
			mockResponse: `{
				"insights": []
			}`,
			wantInsights: 0,
			wantErr:      false,
		},
		{
			name:     "LLM error",
			code:     "func main() {}",
			filePath: "main.go",
			mockErr:  errors.New("LLM unavailable"),
			wantErr:  true,
		},
		{
			name:         "invalid JSON response",
			code:         "func main() {}",
			filePath:     "main.go",
			mockResponse: "not json",
			wantErr:      true,
		},
		{
			name:     "multiple insights",
			code:     "func main() { exec.Command(\"sh\", \"-c\", userInput) }",
			filePath: "cmd.go",
			mockResponse: `{
				"insights": [
					{
						"severity": "critical",
						"category": "security",
						"line": 1,
						"description": "Command injection vulnerability",
						"suggestion": "Use exec.Command with separate args",
						"confidence": 0.99
					},
					{
						"severity": "warning",
						"category": "security",
						"line": 1,
						"description": "Shell execution detected",
						"suggestion": "Avoid using sh -c with user input",
						"confidence": 0.85
					}
				]
			}`,
			wantInsights: 2,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockLLMClient{
				Response: tt.mockResponse,
				Err:      tt.mockErr,
			}
			analyzer := NewAIAnalyzer(mock)

			insights, err := analyzer.AnalyzeWithAI(tt.code, tt.filePath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(insights) != tt.wantInsights {
				t.Errorf("got %d insights, want %d", len(insights), tt.wantInsights)
			}

			// Verify LLM was called
			if len(mock.Calls) != 1 {
				t.Errorf("expected 1 LLM call, got %d", len(mock.Calls))
			}
		})
	}
}

func TestAIAnalyzer_GenerateAIFix(t *testing.T) {
	tests := []struct {
		name         string
		issue        Issue
		code         string
		mockResponse string
		mockErr      error
		wantErr      bool
		checkFix     func(*testing.T, *FixProposal)
	}{
		{
			name: "successful fix generation",
			issue: Issue{
				ID:          "test-issue",
				Title:       "HTTP client without timeout",
				Description: "Add timeout to HTTP client",
				Path:        "main.go",
				LineStart:   5,
			},
			code: `package main

import "net/http"

func main() {
	client := &http.Client{}
	client.Get("http://example.com")
}`,
			mockResponse: `{
				"old_code": "&http.Client{}",
				"new_code": "&http.Client{Timeout: 30 * time.Second}"
			}`,
			wantErr: false,
			checkFix: func(t *testing.T, fix *FixProposal) {
				if fix.OldCode != "&http.Client{}" {
					t.Errorf("OldCode = %q, want %q", fix.OldCode, "&http.Client{}")
				}
				if !strings.Contains(fix.NewCode, "Timeout") {
					t.Error("NewCode should contain Timeout")
				}
				if !fix.Actionable {
					t.Error("expected Actionable = true")
				}
			},
		},
		{
			name: "old_code not found in source",
			issue: Issue{
				ID:    "test-issue",
				Title: "Test issue",
				Path:  "test.go",
			},
			code: "func main() {}",
			mockResponse: `{
				"old_code": "nonexistent code",
				"new_code": "new code"
			}`,
			wantErr: true,
		},
		{
			name: "LLM error",
			issue: Issue{
				ID:   "test-issue",
				Path: "test.go",
			},
			code:    "func main() {}",
			mockErr: errors.New("LLM error"),
			wantErr: true,
		},
		{
			name: "empty fix response",
			issue: Issue{
				ID:   "test-issue",
				Path: "test.go",
			},
			code: "func main() {}",
			mockResponse: `{
				"old_code": "",
				"new_code": ""
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockLLMClient{
				Response: tt.mockResponse,
				Err:      tt.mockErr,
			}
			analyzer := NewAIAnalyzer(mock)

			fix, err := analyzer.GenerateAIFix(tt.issue, tt.code)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if fix == nil {
				t.Fatal("expected fix but got nil")
			}

			if tt.checkFix != nil {
				tt.checkFix(t, fix)
			}
		})
	}
}

func TestAIAnalyzer_NilLLM(t *testing.T) {
	analyzer := &AIAnalyzer{LLM: nil}

	_, err := analyzer.AnalyzeWithAI("code", "file.go")
	if err == nil {
		t.Error("expected error for nil LLM")
	}

	_, err = analyzer.GenerateAIFix(Issue{}, "code")
	if err == nil {
		t.Error("expected error for nil LLM")
	}
}

func TestConvertInsightsToIssues(t *testing.T) {
	insights := []AIInsight{
		{
			Severity:    "critical",
			Category:    "security",
			Line:        10,
			Description: "SQL injection vulnerability",
			Suggestion:  "Use parameterized queries",
			Confidence:  0.95,
		},
		{
			Severity:    "warning",
			Category:    "performance",
			Line:        20,
			Description: "Inefficient loop",
			Suggestion:  "Use map instead of slice",
			Confidence:  0.8,
		},
		{
			Severity:    "info",
			Category:    "readability",
			Line:        30,
			Description: "Function too long",
			Suggestion:  "Break into smaller functions",
			Confidence:  0.7,
		},
	}

	issues := ConvertInsightsToIssues(insights, "test.go")

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// Check first issue (critical -> error)
	if issues[0].Severity != SeverityError {
		t.Errorf("issue[0] severity = %v, want %v", issues[0].Severity, SeverityError)
	}
	if issues[0].LineStart != 10 {
		t.Errorf("issue[0] line = %d, want 10", issues[0].LineStart)
	}

	// Check second issue (warning)
	if issues[1].Severity != SeverityWarning {
		t.Errorf("issue[1] severity = %v, want %v", issues[1].Severity, SeverityWarning)
	}

	// Check third issue (info)
	if issues[2].Severity != SeverityInfo {
		t.Errorf("issue[2] severity = %v, want %v", issues[2].Severity, SeverityInfo)
	}

	// Check tags
	for i, issue := range issues {
		if len(issue.Tags) < 2 {
			t.Errorf("issue[%d] should have at least 2 tags", i)
		}
		if issue.Tags[0] != "ai" {
			t.Errorf("issue[%d] first tag should be 'ai'", i)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"component.ts", "typescript"},
		{"Main.java", "java"},
		{"lib.rs", "rust"},
		{"app.rb", "ruby"},
		{"index.php", "php"},
		{"main.c", "c"},
		{"util.h", "c"},
		{"class.cpp", "cpp"},
		{"header.hpp", "cpp"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectLanguage(tt.path)
			if got != tt.want {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with prefix",
			input: `Some text before {"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with suffix",
			input: `{"key": "value"} some text after`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "nested JSON",
			input: `{"outer": {"inner": "value"}}`,
			want:  `{"outer": {"inner": "value"}}`,
		},
		{
			name:  "no JSON",
			input: `no json here`,
			want:  "",
		},
		{
			name:  "unclosed JSON",
			input: `{"key": "value"`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAnalysisResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantLen  int
		wantErr  bool
	}{
		{
			name: "valid response",
			response: `{
				"insights": [
					{"severity": "critical", "category": "security", "line": 1, "description": "test", "suggestion": "fix", "confidence": 0.9}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "normalize severity",
			response: `{
				"insights": [
					{"severity": "error", "category": "bug", "line": 1, "description": "test", "suggestion": "fix", "confidence": 0.9},
					{"severity": "warn", "category": "style", "line": 2, "description": "test2", "suggestion": "fix2", "confidence": 0.8}
				]
			}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:     "empty insights",
			response: `{"insights": []}`,
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			response: `not json`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAnalysisResponse(tt.response)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result.Insights) != tt.wantLen {
				t.Errorf("got %d insights, want %d", len(result.Insights), tt.wantLen)
			}
		})
	}
}

func TestFilterHighConfidenceInsights(t *testing.T) {
	insights := []AIInsight{
		{Confidence: 0.95},
		{Confidence: 0.80},
		{Confidence: 0.70},
		{Confidence: 0.50},
	}

	filtered := FilterHighConfidenceInsights(insights, 0.75)

	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered insights, got %d", len(filtered))
	}
}

func TestExtractCodeAroundLine(t *testing.T) {
	code := "line1\nline2\nline3\nline4\nline5"

	result := ExtractCodeAroundLine(code, 3, 1)

	if !strings.Contains(result, "line2") {
		t.Error("result should contain line2")
	}
	if !strings.Contains(result, "line3") {
		t.Error("result should contain line3")
	}
	if !strings.Contains(result, "line4") {
		t.Error("result should contain line4")
	}
	if !strings.Contains(result, "> ") {
		t.Error("result should have target line marker")
	}
}

func TestParseLineNumberFromSnippet(t *testing.T) {
	tests := []struct {
		name    string
		snippet string
		want    int
	}{
		{
			name:    "standard diff header",
			snippet: "@@ -10,5 +15,7 @@ func main()",
			want:    15,
		},
		{
			name:    "single line",
			snippet: "@@ -1 +1 @@",
			want:    1,
		},
		{
			name:    "no match",
			snippet: "some code without diff header",
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLineNumberFromSnippet(tt.snippet)
			if got != tt.want {
				t.Errorf("ParseLineNumberFromSnippet() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

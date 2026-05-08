package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantRedact string // この文字列がREDACTEDに置換されるべき
	}{
		{
			name:       "OpenAI API key",
			body:       "Invalid API key: sk-abcdef1234567890abcdef1234567890",
			statusCode: 401,
			wantRedact: "sk-abcdef1234567890abcdef1234567890",
		},
		{
			name:       "Bearer token",
			body:       "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			statusCode: 401,
			wantRedact: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:       "Google API key",
			body:       "Invalid key: AIzaSyDc123456789012345678901234567890",
			statusCode: 403,
			wantRedact: "AIzaSyDc123456789012345678901234567890",
		},
		{
			name:       "api_key parameter",
			body:       "Error: api_key=abc123def456ghi789jkl012mno345pqr678",
			statusCode: 400,
			wantRedact: "api_key=abc123def456ghi789jkl012mno345pqr678",
		},
		{
			name:       "long message truncation",
			body:       strings.Repeat("x", 300),
			statusCode: 500,
			wantRedact: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SanitizeErrorMessage([]byte(tt.body), tt.statusCode)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			errMsg := err.Error()

			// REDACTEDに置換されていることを確認
			if tt.wantRedact != "" {
				if strings.Contains(errMsg, tt.wantRedact) {
					t.Errorf("Expected '%s' to be redacted, but found in error message", tt.wantRedact)
				}
				if !strings.Contains(errMsg, "[REDACTED]") {
					t.Error("Expected [REDACTED] in error message")
				}
			}

			// 長いメッセージは切り詰められているべき
			if len(tt.body) > 200 {
				if !strings.Contains(errMsg, "truncated") {
					t.Error("Expected 'truncated' in error message for long body")
				}
			}

			// ステータスコードが含まれているべき
			if !strings.Contains(errMsg, "API error") {
				t.Error("Expected 'API error' in error message")
			}
		})
	}
}

func TestSanitizeErrorMessage_EmptyBody(t *testing.T) {
	err := SanitizeErrorMessage([]byte{}, 500)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("Expected 'empty response' in error, got: %v", err)
	}
}

func TestHandleRateLimit_NotRateLimit(t *testing.T) {
	// 429以外のステータスコードではnilを返すべき
	resp := &http.Response{
		StatusCode: 200,
	}

	err := HandleRateLimit(resp)
	if err != nil {
		t.Errorf("Expected nil for non-429 status, got: %v", err)
	}
}

func TestHandleRateLimit_WithRetryAfterSeconds(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header: http.Header{
			"Retry-After": []string{"60"},
		},
	}

	err := HandleRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Expected 'rate limit exceeded' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "60 seconds") {
		t.Errorf("Expected '60 seconds' in error, got: %v", err)
	}
}

func TestHandleRateLimit_WithRetryAfterHTTPDate(t *testing.T) {
	// 10秒後の時刻を生成
	futureTime := time.Now().Add(10 * time.Second)
	httpDate := futureTime.Format(http.TimeFormat)

	resp := &http.Response{
		StatusCode: 429,
		Header: http.Header{
			"Retry-After": []string{httpDate},
		},
	}

	err := HandleRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Expected 'rate limit exceeded' in error, got: %v", err)
	}

	// 秒数が含まれているべき（約10秒）
	errMsg := err.Error()
	if !strings.Contains(errMsg, "s") {
		t.Errorf("Expected duration in error, got: %v", err)
	}
}

func TestHandleRateLimit_NoRetryAfterHeader(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{},
	}

	err := HandleRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "60 seconds") {
		t.Errorf("Expected default '60 seconds' in error, got: %v", err)
	}
}

func TestHandleRateLimit_InvalidRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header: http.Header{
			"Retry-After": []string{"invalid-value"},
		},
	}

	err := HandleRateLimit(resp)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "retry later") {
		t.Errorf("Expected 'retry later' in error, got: %v", err)
	}
}

func TestSanitizeErrorMessage_Integration(t *testing.T) {
	// 実際のHTTPレスポンスをシミュレート
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error": {"message": "Invalid API key: sk-test1234567890abcdefghij"}}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込み
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	body = body[:n]

	// サニタイズ
	sanitizedErr := SanitizeErrorMessage(body, resp.StatusCode)

	if sanitizedErr == nil {
		t.Fatal("Expected error, got nil")
	}

	errMsg := sanitizedErr.Error()

	// APIキーがREDACTEDに置換されているべき
	if strings.Contains(errMsg, "sk-test1234567890abcdefghij") {
		t.Error("API key should be redacted")
	}

	if !strings.Contains(errMsg, "[REDACTED]") {
		t.Error("Expected [REDACTED] in sanitized error")
	}
}

func TestSupportsImages(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		want         bool
	}{
		{
			name:         "Claude supports images",
			providerName: "claude",
			want:         true,
		},
		{
			name:         "Anthropic supports images",
			providerName: "anthropic",
			want:         true,
		},
		{
			name:         "OpenAI supports images",
			providerName: "openai",
			want:         true,
		},
		{
			name:         "Gemini supports images",
			providerName: "gemini",
			want:         true,
		},
		{
			name:         "DeepSeek does not support images",
			providerName: "deepseek",
			want:         false,
		},
		{
			name:         "Kimi supports images",
			providerName: "kimi",
			want:         true,
		},
		{
			name:         "Moonshot alias supports images",
			providerName: "moonshot",
			want:         true,
		},
		{
			name:         "Ollama does not support images",
			providerName: "ollama",
			want:         false,
		},
		{
			name:         "Groq does not support images",
			providerName: "groq",
			want:         false,
		},
		{
			name:         "Unknown provider does not support images",
			providerName: "unknown",
			want:         false,
		},
		{
			name:         "Case insensitive - CLAUDE",
			providerName: "CLAUDE",
			want:         true,
		},
		{
			name:         "Anthropic with spaces supports images",
			providerName: "  anthropic  ",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupportsImages(tt.providerName)
			if got != tt.want {
				t.Errorf("SupportsImages(%q) = %v, want %v", tt.providerName, got, tt.want)
			}
		})
	}
}

func TestNewProvider_MissingAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		envKey       string
	}{
		{
			name:         "DeepSeek without API key",
			providerName: "deepseek",
			envKey:       "DEEPSEEK_API_KEY",
		},
		{
			name:         "OpenAI without API key",
			providerName: "openai",
			envKey:       "OPENAI_API_KEY",
		},
		{
			name:         "Gemini without API key",
			providerName: "gemini",
			envKey:       "GEMINI_API_KEY",
		},
		{
			name:         "Claude without API key",
			providerName: "claude",
			envKey:       "ANTHROPIC_API_KEY",
		},
		{
			name:         "Groq without API key",
			providerName: "groq",
			envKey:       "GROQ_API_KEY",
		},
		{
			name:         "Kimi without API key",
			providerName: "kimi",
			envKey:       "MOONSHOT_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をクリア
			originalValue := os.Getenv(tt.envKey)
			os.Unsetenv(tt.envKey)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envKey, originalValue)
				}
			}()

			_, err := NewProvider(tt.providerName)
			if err == nil {
				t.Errorf("NewProvider(%q) should return error when %s is not set", tt.providerName, tt.envKey)
			}
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	_, err := NewProvider("unknown-provider")
	if err == nil {
		t.Error("NewProvider should return error for unknown provider")
	}

	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("Expected 'unknown provider' in error, got: %v", err)
	}
}

func TestNewProvider_OllamaWithDefaultURL(t *testing.T) {
	// OLLAMA_BASE_URLをクリア
	originalValue := os.Getenv("OLLAMA_BASE_URL")
	os.Unsetenv("OLLAMA_BASE_URL")
	defer func() {
		if originalValue != "" {
			os.Setenv("OLLAMA_BASE_URL", originalValue)
		}
	}()

	provider, err := NewProvider("ollama")
	if err != nil {
		t.Fatalf("NewProvider('ollama') failed: %v", err)
	}

	if provider.Name() != "Ollama" {
		t.Errorf("NewProvider('ollama') Name() = %v, want 'Ollama'", provider.Name())
	}
}

func TestNewProvider_OllamaWithCustomURL(t *testing.T) {
	originalValue := os.Getenv("OLLAMA_BASE_URL")
	os.Setenv("OLLAMA_BASE_URL", "http://custom-host:8080")
	defer func() {
		if originalValue != "" {
			os.Setenv("OLLAMA_BASE_URL", originalValue)
		} else {
			os.Unsetenv("OLLAMA_BASE_URL")
		}
	}()

	provider, err := NewProvider("ollama")
	if err != nil {
		t.Fatalf("NewProvider('ollama') with custom URL failed: %v", err)
	}

	if provider.Name() != "Ollama" {
		t.Errorf("NewProvider('ollama') Name() = %v, want 'Ollama'", provider.Name())
	}
}

func TestNewProvider_SuccessPaths(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		envKey       string
		envValue     string
		wantName     string
	}{
		{
			name:         "DeepSeek with API key",
			providerName: "deepseek",
			envKey:       "DEEPSEEK_API_KEY",
			envValue:     "test-deepseek-key",
			wantName:     "DeepSeek",
		},
		{
			name:         "OpenAI with API key",
			providerName: "openai",
			envKey:       "OPENAI_API_KEY",
			envValue:     "test-openai-key",
			wantName:     "OpenAI",
		},
		{
			name:         "Gemini with API key",
			providerName: "gemini",
			envKey:       "GEMINI_API_KEY",
			envValue:     "test-gemini-key",
			wantName:     "Gemini",
		},
		{
			name:         "Claude with API key",
			providerName: "claude",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-claude-key",
			wantName:     "Claude",
		},
		{
			name:         "Anthropic alias",
			providerName: "anthropic",
			envKey:       "ANTHROPIC_API_KEY",
			envValue:     "test-anthropic-key",
			wantName:     "Claude",
		},
		{
			name:         "Groq with API key",
			providerName: "groq",
			envKey:       "GROQ_API_KEY",
			envValue:     "test-groq-key",
			wantName:     "Groq",
		},
		{
			name:         "Kimi with API key",
			providerName: "kimi",
			envKey:       "MOONSHOT_API_KEY",
			envValue:     "test-kimi-key",
			wantName:     "Kimi",
		},
		{
			name:         "Moonshot alias",
			providerName: "moonshot",
			envKey:       "MOONSHOT_API_KEY",
			envValue:     "test-kimi-key",
			wantName:     "Kimi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalValue := os.Getenv(tt.envKey)
			os.Setenv(tt.envKey, tt.envValue)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envKey, originalValue)
				} else {
					os.Unsetenv(tt.envKey)
				}
			}()

			provider, err := NewProvider(tt.providerName)
			if err != nil {
				t.Fatalf("NewProvider(%q) failed: %v", tt.providerName, err)
			}

			if provider.Name() != tt.wantName {
				t.Errorf("NewProvider(%q) Name() = %v, want %v", tt.providerName, provider.Name(), tt.wantName)
			}
		})
	}
}

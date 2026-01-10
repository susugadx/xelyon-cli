package api

import (
	"net/http"
	"net/http/httptest"
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
			err := sanitizeErrorMessage([]byte(tt.body), tt.statusCode)

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
	err := sanitizeErrorMessage([]byte{}, 500)

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

	err := handleRateLimit(resp)
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

	err := handleRateLimit(resp)

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

	err := handleRateLimit(resp)

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

	err := handleRateLimit(resp)

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

	err := handleRateLimit(resp)

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
		w.Write([]byte(`{"error": {"message": "Invalid API key: sk-test1234567890abcdefghij"}}`))
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
	sanitizedErr := sanitizeErrorMessage(body, resp.StatusCode)

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

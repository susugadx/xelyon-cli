package tools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRequestGET(t *testing.T) {
	// テストサーバーを起動
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	result, err := executeHTTPRequest("GET", server.URL, "", "", 30)
	if err != nil {
		t.Fatalf("executeHTTPRequest failed: %v", err)
	}

	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200 in result, got: %s", result)
	}
	if !strings.Contains(result, "success") {
		t.Errorf("Expected 'success' in result, got: %s", result)
	}
}

func TestHTTPRequestPOST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 1, "created": true}`))
	}))
	defer server.Close()

	body := `{"name": "test"}`
	result, err := executeHTTPRequest("POST", server.URL, "", body, 30)
	if err != nil {
		t.Fatalf("executeHTTPRequest failed: %v", err)
	}

	if !strings.Contains(result, "201") {
		t.Errorf("Expected status 201 in result, got: %s", result)
	}
}

func TestHTTPRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected X-Custom-Header, got %s", r.Header.Get("X-Custom-Header"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	headers := `{"Authorization": "Bearer test-token", "X-Custom-Header": "custom-value"}`
	result, err := executeHTTPRequest("GET", server.URL, headers, "", 30)
	if err != nil {
		t.Fatalf("executeHTTPRequest failed: %v", err)
	}

	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200 in result, got: %s", result)
	}
}

func TestHTTPRequestEmptyURL(t *testing.T) {
	_, err := executeHTTPRequest("GET", "", "", "", 30)
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}

func TestHTTPRequestInvalidMethod(t *testing.T) {
	_, err := executeHTTPRequest("INVALID", "https://example.com", "", "", 30)
	if err == nil {
		t.Error("Expected error for invalid HTTP method")
	}
}

func TestHTTPRequestInvalidHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := executeHTTPRequest("GET", server.URL, "not-valid-json", "", 30)
	if err == nil {
		t.Error("Expected error for invalid headers JSON")
	}
}

func TestHTTPRequestDefaultMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET as default method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// メソッド空文字でデフォルトGET
	result, err := executeHTTPRequest("", server.URL, "", "", 30)
	if err != nil {
		t.Fatalf("executeHTTPRequest failed: %v", err)
	}

	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200, got: %s", result)
	}
}

func TestHTTPRequestJSONFormatting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Pretty印刷可能なJSONを返す
		response := map[string]interface{}{
			"users": []map[string]string{
				{"name": "Alice", "email": "alice@example.com"},
				{"name": "Bob", "email": "bob@example.com"},
			},
		}
		jsonData, _ := json.Marshal(response)
		_, _ = w.Write(jsonData)
	}))
	defer server.Close()

	result, err := executeHTTPRequest("GET", server.URL, "", "", 30)
	if err != nil {
		t.Fatalf("executeHTTPRequest failed: %v", err)
	}

	// フォーマットされたJSONが含まれているべき
	if !strings.Contains(result, "Alice") {
		t.Errorf("Expected 'Alice' in result, got: %s", result)
	}
}

func TestHTTPRequestTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	tool := &HTTPRequestTool{}

	if tool.Name() != "http_request" {
		t.Errorf("Expected tool name 'http_request', got '%s'", tool.Name())
	}

	result, change, err := tool.Run(map[string]string{
		"method": "GET",
		"url":    server.URL,
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	// http_requestはファイル変更しないのでchangeはnil
	if change != nil {
		t.Error("Expected no FileChange for http_request")
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}

package serper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mockAPIServer creates a test HTTP server
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// assertRequestMethod verifies the HTTP method
func assertRequestMethod(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != expected {
		t.Errorf("Expected method %s, got %s", expected, r.Method)
	}
}

// assertJSONContentType verifies the Content-Type header
func assertJSONContentType(t *testing.T, r *http.Request) {
	t.Helper()
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

// assertRequestHeader verifies a specific header
func assertRequestHeader(t *testing.T, r *http.Request, key, expected string) {
	t.Helper()
	value := r.Header.Get(key)
	if value != expected {
		t.Errorf("Expected header %s=%s, got %s", key, expected, value)
	}
}

func TestWebSearch_Success(t *testing.T) {
	// Create mock server
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)
		assertRequestHeader(t, r, "X-API-KEY", "test-api-key")

		// Return mock response
		response := Response{
			Organic: []SearchResult{
				{Title: "Result 1", Link: "https://example.com/1", Snippet: "Snippet 1"},
				{Title: "Result 2", Link: "https://example.com/2", Snippet: "Snippet 2"},
				{Title: "Result 3", Link: "https://example.com/3", Snippet: "Snippet 3"},
				{Title: "Result 4", Link: "https://example.com/4", Snippet: "Snippet 4"},
				{Title: "Result 5", Link: "https://example.com/5", Snippet: "Snippet 5"},
				{Title: "Result 6", Link: "https://example.com/6", Snippet: "Snippet 6"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variables
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-api-key")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
		os.Unsetenv("SERPER_API_KEY")
	})

	// Execute
	result, err := WebSearch("test query")

	// Verify
	if err != nil {
		t.Fatalf("WebSearch() error = %v", err)
	}

	// Should contain result count
	if !strings.Contains(result, "Found 6 results") {
		t.Errorf("Result should contain result count, got: %s", result)
	}

	// Should contain first 5 results (max)
	if !strings.Contains(result, "Result 1") {
		t.Error("Result should contain 'Result 1'")
	}
	if !strings.Contains(result, "Result 5") {
		t.Error("Result should contain 'Result 5'")
	}
	// Result 6 should also be mentioned in count but not listed (only top 5)
	if strings.Contains(result, "Result 6") {
		t.Error("Result should not contain 'Result 6' (only top 5)")
	}
}

func TestWebSearch_NoAPIKey(t *testing.T) {
	// Ensure no API key is set
	os.Unsetenv("SERPER_API_KEY")

	// Execute
	result, err := WebSearch("test query")

	// Verify
	if err == nil {
		t.Fatal("WebSearch() should return error when API key is not set")
	}

	if !strings.Contains(err.Error(), "SERPER_API_KEY environment variable is not set") {
		t.Errorf("Error should mention SERPER_API_KEY, got: %v", err)
	}

	if result != "" {
		t.Errorf("Result should be empty, got: %s", result)
	}
}

func TestWebSearch_EmptyResults(t *testing.T) {
	// Create mock server returning empty results
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Organic: []SearchResult{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variables
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-api-key")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
		os.Unsetenv("SERPER_API_KEY")
	})

	// Execute
	result, err := WebSearch("no results query")

	// Verify
	if err != nil {
		t.Fatalf("WebSearch() error = %v", err)
	}

	if result != "No results found." {
		t.Errorf("Result should be 'No results found.', got: %s", result)
	}
}

func TestWebSearch_APIError(t *testing.T) {
	// Create mock server returning error
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
	})

	// Set environment variables
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-api-key")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
		os.Unsetenv("SERPER_API_KEY")
	})

	// Execute
	result, err := WebSearch("test query")

	// Verify
	if err == nil {
		t.Fatal("WebSearch() should return error on API error")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should contain status code 500, got: %v", err)
	}

	if result != "" {
		t.Errorf("Result should be empty, got: %s", result)
	}
}

func TestWebSearch_InvalidJSON(t *testing.T) {
	// Create mock server returning invalid JSON
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`invalid json`))
	})

	// Set environment variables
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-api-key")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
		os.Unsetenv("SERPER_API_KEY")
	})

	// Execute
	result, err := WebSearch("test query")

	// Verify
	if err == nil {
		t.Fatal("WebSearch() should return error on invalid JSON")
	}

	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Error should mention unmarshal failure, got: %v", err)
	}

	if result != "" {
		t.Errorf("Result should be empty, got: %s", result)
	}
}

func TestWebSearch_LessThan5Results(t *testing.T) {
	// Create mock server returning 3 results
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Organic: []SearchResult{
				{Title: "Result 1", Link: "https://example.com/1", Snippet: "Snippet 1"},
				{Title: "Result 2", Link: "https://example.com/2", Snippet: "Snippet 2"},
				{Title: "Result 3", Link: "https://example.com/3", Snippet: "Snippet 3"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variables
	os.Setenv("SERPER_API_URL", server.URL)
	os.Setenv("SERPER_API_KEY", "test-api-key")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
		os.Unsetenv("SERPER_API_KEY")
	})

	// Execute
	result, err := WebSearch("test query")

	// Verify
	if err != nil {
		t.Fatalf("WebSearch() error = %v", err)
	}

	// Should contain all 3 results
	if !strings.Contains(result, "Found 3 results") {
		t.Errorf("Result should contain 'Found 3 results', got: %s", result)
	}

	if !strings.Contains(result, "Result 1") {
		t.Error("Result should contain 'Result 1'")
	}
	if !strings.Contains(result, "Result 2") {
		t.Error("Result should contain 'Result 2'")
	}
	if !strings.Contains(result, "Result 3") {
		t.Error("Result should contain 'Result 3'")
	}
}

func TestGetURL_Default(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("SERPER_API_URL")

	url := getURL()

	if url != "https://google.serper.dev/search" {
		t.Errorf("getURL() = %q, want default URL", url)
	}
}

func TestGetURL_Override(t *testing.T) {
	os.Setenv("SERPER_API_URL", "https://custom.example.com/search")
	t.Cleanup(func() {
		os.Unsetenv("SERPER_API_URL")
	})

	url := getURL()

	if url != "https://custom.example.com/search" {
		t.Errorf("getURL() = %q, want custom URL", url)
	}
}

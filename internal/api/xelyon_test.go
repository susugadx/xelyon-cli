package api

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestSearchRAG_Success(t *testing.T) {
	// Create mock server
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assertRequestMethod(t, r, "POST")
		assertJSONContentType(t, r)

		// Verify request path
		if r.URL.Path != "/api/rag/search" {
			t.Errorf("Request path = %q, want /api/rag/search", r.URL.Path)
		}

		// Return mock response
		response := RAGResponse{
			Results: []RAGResult{
				{
					ID:            "1",
					Content:       "Test content 1",
					DocumentID:    "doc1",
					DocumentTitle: "Document 1",
					DocumentType:  "text",
					Similarity:    0.95,
				},
				{
					ID:            "2",
					Content:       "Test content 2",
					DocumentID:    "doc2",
					DocumentTitle: "Document 2",
					DocumentType:  "text",
					Similarity:    0.85,
				},
			},
			Query: "test query",
			Count: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variable
	os.Setenv("XELYON_API_URL", server.URL)
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	// Execute
	result, err := SearchRAG("test query", "user123", 10)

	// Verify
	if err != nil {
		t.Fatalf("SearchRAG() error = %v", err)
	}

	if result == nil {
		t.Fatal("SearchRAG() returned nil result")
	}

	if result.Count != 2 {
		t.Errorf("Result.Count = %d, want 2", result.Count)
	}

	if len(result.Results) != 2 {
		t.Errorf("len(Result.Results) = %d, want 2", len(result.Results))
	}

	if result.Query != "test query" {
		t.Errorf("Result.Query = %q, want 'test query'", result.Query)
	}

	// Verify first result
	if result.Results[0].ID != "1" {
		t.Errorf("Results[0].ID = %q, want '1'", result.Results[0].ID)
	}
	if result.Results[0].Similarity != 0.95 {
		t.Errorf("Results[0].Similarity = %f, want 0.95", result.Results[0].Similarity)
	}
}

func TestSearchRAG_EmptyResults(t *testing.T) {
	// Create mock server returning empty results
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := RAGResponse{
			Results: []RAGResult{},
			Query:   "no results query",
			Count:   0,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variable
	os.Setenv("XELYON_API_URL", server.URL)
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	// Execute
	result, err := SearchRAG("no results query", "user123", 10)

	// Verify
	if err != nil {
		t.Fatalf("SearchRAG() error = %v", err)
	}

	if result == nil {
		t.Fatal("SearchRAG() returned nil result")
	}

	if result.Count != 0 {
		t.Errorf("Result.Count = %d, want 0", result.Count)
	}

	if len(result.Results) != 0 {
		t.Errorf("len(Result.Results) = %d, want 0", len(result.Results))
	}
}

func TestSearchRAG_APIError(t *testing.T) {
	// Create mock server returning error
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
	})

	// Set environment variable
	os.Setenv("XELYON_API_URL", server.URL)
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	// Execute
	result, err := SearchRAG("test query", "user123", 10)

	// Verify
	if err == nil {
		t.Fatal("SearchRAG() should return error on API error")
	}

	if result != nil {
		t.Errorf("Result should be nil on error, got %v", result)
	}
}

func TestSearchRAG_InvalidJSON(t *testing.T) {
	// Create mock server returning invalid JSON
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	})

	// Set environment variable
	os.Setenv("XELYON_API_URL", server.URL)
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	// Execute
	result, err := SearchRAG("test query", "user123", 10)

	// Verify
	if err == nil {
		t.Fatal("SearchRAG() should return error on invalid JSON")
	}

	if result != nil {
		t.Errorf("Result should be nil on error, got %v", result)
	}
}

func TestSearchRAG_RequestBody(t *testing.T) {
	// Capture the request body to verify it's correct
	var capturedRequest RAGRequest

	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Decode request body
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Return success response
		response := RAGResponse{Results: []RAGResult{}, Query: "test", Count: 0}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Set environment variable
	os.Setenv("XELYON_API_URL", server.URL)
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	// Execute
	_, err := SearchRAG("test query", "user456", 5)

	// Verify
	if err != nil {
		t.Fatalf("SearchRAG() error = %v", err)
	}

	// Verify request body
	if capturedRequest.Query != "test query" {
		t.Errorf("Request.Query = %q, want 'test query'", capturedRequest.Query)
	}
	if capturedRequest.UserID != "user456" {
		t.Errorf("Request.UserID = %q, want 'user456'", capturedRequest.UserID)
	}
	if capturedRequest.TopK != 5 {
		t.Errorf("Request.TopK = %d, want 5", capturedRequest.TopK)
	}
	if capturedRequest.Model != "small" {
		t.Errorf("Request.Model = %q, want 'small'", capturedRequest.Model)
	}
}

func TestGetXelyonURL_Default(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("XELYON_API_URL")

	url := getXelyonURL()

	if url != defaultXelyonURL {
		t.Errorf("getXelyonURL() = %q, want default URL", url)
	}
}

func TestGetXelyonURL_Override(t *testing.T) {
	os.Setenv("XELYON_API_URL", "https://custom.example.com")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_API_URL")
	})

	url := getXelyonURL()

	if url != "https://custom.example.com" {
		t.Errorf("getXelyonURL() = %q, want custom URL", url)
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockAPIServer creates a test server with the given handler
// The server is automatically closed when the test finishes
func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// jsonHandler returns an HTTP handler that responds with JSON
func jsonHandler(statusCode int, body interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

// errorHandler returns an HTTP handler that responds with an error
func errorHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		errResp := map[string]interface{}{
			"error": map[string]string{
				"message": message,
			},
		}
		json.NewEncoder(w).Encode(errResp)
	}
}

// streamingHandler returns an HTTP handler that streams OpenAI-style responses
func streamingHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

// rateLimitHandler returns an HTTP handler that returns a 429 rate limit response
func rateLimitHandler(retryAfter string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
		errResp := map[string]interface{}{
			"error": map[string]string{
				"message": "Rate limit exceeded",
			},
		}
		json.NewEncoder(w).Encode(errResp)
	}
}

// delayHandler returns an HTTP handler that delays before responding
// Useful for testing timeout scenarios
func delayHandler(delayDuration int, statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		default:
			// Simulate delay by blocking (in real tests, use time.Sleep)
			w.WriteHeader(statusCode)
		}
	}
}

// assertRequestMethod checks that the request has the expected method
func assertRequestMethod(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != expected {
		t.Errorf("Request method = %s, want %s", r.Method, expected)
	}
}

// assertRequestHeader checks that the request has the expected header value
func assertRequestHeader(t *testing.T, r *http.Request, header, expected string) {
	t.Helper()
	if got := r.Header.Get(header); got != expected {
		t.Errorf("Request header %s = %q, want %q", header, got, expected)
	}
}

// assertContentType checks that the request has JSON content type
func assertJSONContentType(t *testing.T, r *http.Request) {
	t.Helper()
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

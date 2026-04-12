package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProvider_CompactHistory_DefaultModelAndResponseParsing(t *testing.T) {
	var captured CompactRequest
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"gpt-5.4-mini",
			"output":[{"type":"compacted","data":"opaque"}],
			"usage":{"input_tokens":12,"output_tokens":4,"total_tokens":16}
		}`))
	})

	originalURL := os.Getenv("OPENAI_COMPACT_URL")
	t.Cleanup(func() {
		if originalURL == "" {
			os.Unsetenv("OPENAI_COMPACT_URL")
		} else {
			_ = os.Setenv("OPENAI_COMPACT_URL", originalURL)
		}
	})
	_ = os.Setenv("OPENAI_COMPACT_URL", server.URL)

	p := New("test-key")
	resp, err := p.CompactHistory(context.Background(), []api.InputItem{{Type: "message", Role: "user", Content: "hello"}}, "", "summarize")
	if err != nil {
		t.Fatalf("CompactHistory() error = %v", err)
	}
	if captured.Model != "gpt-5.4" {
		t.Fatalf("request model = %q, want %q", captured.Model, "gpt-5.4")
	}
	if captured.Instructions != "summarize" {
		t.Fatalf("Instructions = %q, want %q", captured.Instructions, "summarize")
	}
	if resp.Model != "gpt-5.4-mini" || len(resp.Output) != 1 || resp.Output[0].Data != "opaque" {
		t.Fatalf("CompactHistory() response = %+v", resp)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 16 {
		t.Fatalf("CompactHistory() usage = %+v, want total=16", resp.Usage)
	}
}

func TestProvider_CompactHistory_HTTPError(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream unavailable"}}`))
	})
	t.Setenv("OPENAI_COMPACT_URL", server.URL)

	p := New("test-key")
	_, err := p.CompactHistory(context.Background(), nil, "gpt-5.4-mini", "")
	if err == nil {
		t.Fatal("CompactHistory() should return error for non-200 response")
	}
}

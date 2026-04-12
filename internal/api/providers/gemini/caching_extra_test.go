package gemini

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDeleteCachedContent_SuccessAndError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var captured *http.Request
		p := New("test-key")
		p.httpClient = &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				captured = req
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}),
		}

		if err := p.DeleteCachedContent(context.Background(), "cachedContents/123"); err != nil {
			t.Fatalf("DeleteCachedContent() error = %v", err)
		}
		if captured == nil {
			t.Fatal("DeleteCachedContent() should issue an HTTP request")
		}
		if captured.Method != http.MethodDelete {
			t.Fatalf("method = %q, want DELETE", captured.Method)
		}
		if captured.URL.String() != "https://generativelanguage.googleapis.com/v1beta/cachedContents/123" {
			t.Fatalf("url = %q, want cached content endpoint", captured.URL.String())
		}
		if captured.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatalf("x-goog-api-key = %q, want test-key", captured.Header.Get("x-goog-api-key"))
		}
	})

	t.Run("api error", func(t *testing.T) {
		p := New("test-key")
		p.httpClient = &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("backend unavailable")),
				}, nil
			}),
		}

		err := p.DeleteCachedContent(context.Background(), "cachedContents/123")
		if err == nil {
			t.Fatal("DeleteCachedContent() should return error for non-2xx status")
		}
		if !strings.Contains(err.Error(), "backend unavailable") {
			t.Fatalf("DeleteCachedContent() error = %q, want backend unavailable", err.Error())
		}
	})
}

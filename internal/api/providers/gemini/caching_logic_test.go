package gemini

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestEstimateTokens(t *testing.T) {
	systemPrompt := "You are a helpful assistant."
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	// "You are a helpful assistant." (28) + "Hello" (5) + "Hi there!" (9) = 42
	expected := 42
	actual := estimateTokens(systemPrompt, history)

	if actual != expected {
		t.Errorf("expected %d tokens, got %d", expected, actual)
	}
}

func TestUpdateOrUseCache(t *testing.T) {
	// テスト用の閾値設定
	longContent := strings.Repeat("a", minCacheTokens+100)

	tests := []struct {
		name             string
		systemPrompt     string
		history          []api.Message
		activeCacheName  string
		cachedMsgCount   int
		expectCache      bool
		expectCreateCall bool
		expectDeleteCall bool
		expectDiffOnly   bool
	}{
		{
			name:         "Small content - No cache",
			systemPrompt: "sys",
			history:      []api.Message{{Role: "user", Content: "hi"}},
			expectCache:  false,
		},
		{
			name:             "Large content - Create cache",
			systemPrompt:     "sys",
			history:          []api.Message{{Role: "user", Content: longContent}, {Role: "user", Content: "last"}},
			expectCache:      true,
			expectCreateCall: true,
			expectDiffOnly:   true,
		},
		{
			name:             "Use existing cache - Small diff",
			systemPrompt:     "sys",
			history:          []api.Message{{Role: "user", Content: longContent}, {Role: "user", Content: "last"}, {Role: "assistant", Content: "res"}, {Role: "user", Content: "next"}},
			activeCacheName:  "cachedContents/old",
			cachedMsgCount:   2,
			expectCache:      true,
			expectCreateCall: false,
			expectDiffOnly:   true,
		},
		{
			name:             "Large diff - Recreate cache",
			systemPrompt:     "sys",
			history:          makeHistory(longContent, 2+maxDiffMessages+1),
			activeCacheName:  "cachedContents/old",
			cachedMsgCount:   2,
			expectCache:      true,
			expectCreateCall: true,
			expectDeleteCall: true,
			expectDiffOnly:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバー設定（実際には使われないが念のため）
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			defer server.Close()

			p := New("test-key")
			p.httpClient = &http.Client{
				Transport: &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						if req.Method == "POST" && strings.Contains(req.URL.Path, "cachedContents") {
							if !tt.expectCreateCall {
								return nil, fmt.Errorf("unexpected CreateCachedContent call")
							}
							respBody := `{"name": "cachedContents/new", "createTime": "2024-01-01T00:00:00Z"}`
							return &http.Response{
								StatusCode: 201,
								Body:       io.NopCloser(strings.NewReader(respBody)),
							}, nil
						}
						if req.Method == "DELETE" && strings.Contains(req.URL.Path, "cachedContents") {
							if !tt.expectDeleteCall {
								return nil, fmt.Errorf("unexpected DeleteCachedContent call")
							}
							return &http.Response{
								StatusCode: 204,
								Body:       io.NopCloser(strings.NewReader("")),
							}, nil
						}
						return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					},
				},
			}

			p.activeCacheName = tt.activeCacheName
			p.cachedMessageCount = tt.cachedMsgCount
			if tt.activeCacheName != "" {
				p.cacheExpireTime = time.Now().Add(1 * time.Hour)
			}

			os.Setenv("GEMINI_CONTEXT_CACHING", "1")
			defer os.Unsetenv("GEMINI_CONTEXT_CACHING")

			cacheName, msgs, err := p.updateOrUseCache(context.Background(), tt.systemPrompt, tt.history, "gemini-1.5-pro", nil, nil)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectCache {
				if cacheName == "" {
					t.Error("expected cache usage, but got empty cacheName")
				}
			} else {
				if cacheName != "" {
					t.Errorf("expected no cache usage, but got %s", cacheName)
				}
			}

			if tt.expectDiffOnly {
				if len(msgs) == len(tt.history) {
					t.Error("expected diff messages, but got full history")
				}
			} else {
				if len(msgs) != len(tt.history) {
					t.Errorf("expected full history (%d), but got %d", len(tt.history), len(msgs))
				}
			}
		})
	}
}

func makeHistory(longContent string, count int) []api.Message {
	hist := make([]api.Message, count)
	hist[0] = api.Message{Role: "user", Content: longContent}
	for i := 1; i < count; i++ {
		hist[i] = api.Message{Role: "user", Content: "msg"}
	}
	return hist
}

type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

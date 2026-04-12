package gemini

import (
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestChatWithTextMode_EmptyErrorBodyIncludesModelName(t *testing.T) {
	server := mockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	t.Setenv("GEMINI_API_URL", server.URL)
	t.Setenv("GEMINI_CONTEXT_CACHING", "0")

	p := New("test-key")
	_, err := p.chatWithTextMode(newGeminiRequestContext(false, "medium"), "System prompt", []api.Message{{Role: "user", Content: "hello"}}, "gemini-2.5-flash")
	if err == nil {
		t.Fatal("chatWithTextMode() should return error for empty non-200 response")
	}
	if !strings.Contains(err.Error(), "empty response body") || !strings.Contains(err.Error(), "gemini-2.5-flash") {
		t.Fatalf("chatWithTextMode() error = %q, want empty response body with model name", err.Error())
	}
}

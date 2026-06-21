package openaisubscription

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func mockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func newOpenAITestContext(t *testing.T, thinking bool) context.Context {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"
	runtime := uiruntime.NewRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	ctx = config.WithContext(ctx, cfg)
	return api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
}

package openaicompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type fakeChatCompletionsExecutor struct {
	resp *http.Response
	err  error
}

func (f *fakeChatCompletionsExecutor) ExecuteRequest(*http.Request) (*http.Response, error) {
	return f.resp, f.err
}

func (f *fakeChatCompletionsExecutor) Name() string {
	return "Fake"
}

func newRunnerTestContext() context.Context {
	runtime := uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	ctx := uiruntime.WithRuntime(context.Background(), runtime)
	return api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
}

func TestRunChatCompletions_UsesStreamHandlerForEventStream(t *testing.T) {
	executor := &fakeChatCompletionsExecutor{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("")),
		},
	}
	req, err := http.NewRequestWithContext(newRunnerTestContext(), http.MethodPost, "https://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	got, err := RunChatCompletions(req.Context(), executor, req, ChatCompletionsRunOptions{
		StreamHandler: func(context.Context, *http.Response, *uiruntime.Spinner) (string, error) {
			return "stream", nil
		},
		NonStreamHandler: func(context.Context, *http.Response, *uiruntime.Spinner) (string, error) {
			t.Fatal("non-stream handler should not be called")
			return "", nil
		},
	})
	if err != nil {
		t.Fatalf("RunChatCompletions() error = %v", err)
	}
	if got != "stream" {
		t.Fatalf("RunChatCompletions() = %q, want stream", got)
	}
}

func TestRunChatCompletions_PrefixesRequestError(t *testing.T) {
	executor := &fakeChatCompletionsExecutor{err: errors.New("network down")}
	req, err := http.NewRequestWithContext(newRunnerTestContext(), http.MethodPost, "https://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	_, err = RunChatCompletions(req.Context(), executor, req, ChatCompletionsRunOptions{
		RequestErrorPrefix: "Provider request failed",
		StreamHandler: func(context.Context, *http.Response, *uiruntime.Spinner) (string, error) {
			return "", nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Provider request failed: network down") {
		t.Fatalf("RunChatCompletions() error = %v, want prefixed network error", err)
	}
}

func TestSimpleProviderSetMCPToolsAndMCPTools(t *testing.T) {
	provider := NewSimpleProvider("test-key", SimpleProviderSpec{
		DisplayName: "Compat",
		DefaultURL:  "https://example.test/v1/chat/completions",
	})
	tools := []api.ToolDefinition{{
		Name:        "mcp_github_get_issue",
		Description: "Get a GitHub issue",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "string"},
			},
		},
	}}

	provider.SetMCPTools(tools)
	got := provider.MCPTools()
	if len(got) != 1 {
		t.Fatalf("len(MCPTools()) = %d, want 1", len(got))
	}
	if got[0].Name != "mcp_github_get_issue" || got[0].Description != "Get a GitHub issue" {
		t.Fatalf("MCPTools()[0] = %#v, want stored MCP definition", got[0])
	}
	if got[0].Parameters["type"] != "object" {
		t.Fatalf("MCPTools()[0].Parameters = %#v, want object schema", got[0].Parameters)
	}

	got[0].Name = "mutated"
	again := provider.MCPTools()
	if again[0].Name != "mcp_github_get_issue" {
		t.Fatalf("MCPTools() returned mutable slice alias: %#v", again)
	}
}

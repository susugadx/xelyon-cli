package openaicompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
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
	runtime := ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)
	ctx := ui.WithRuntime(context.Background(), runtime)
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
		StreamHandler: func(context.Context, *http.Response, *ui.Spinner) (string, error) {
			return "stream", nil
		},
		NonStreamHandler: func(context.Context, *http.Response, *ui.Spinner) (string, error) {
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
		StreamHandler: func(context.Context, *http.Response, *ui.Spinner) (string, error) {
			return "", nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Provider request failed: network down") {
		t.Fatalf("RunChatCompletions() error = %v, want prefixed network error", err)
	}
}

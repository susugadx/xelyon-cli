package openaicompat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// ChatCompletionsExecutor は OpenAI 互換 request を送信できる provider の最小契約。
type ChatCompletionsExecutor interface {
	ExecuteRequest(*http.Request) (*http.Response, error)
	Name() string
}

// ChatCompletionsRunOptions は OpenAI 互換 Chat Completions 実行時の差分処理を表す。
type ChatCompletionsRunOptions struct {
	ImageMode          bool
	SpinnerSuffix      string
	ForceStreaming     bool
	RequestErrorPrefix string
	StreamHandler      func(context.Context, *http.Response, *uiruntime.Spinner) (string, error)
	NonStreamHandler   func(context.Context, *http.Response, *uiruntime.Spinner) (string, error)
}

// RunChatCompletions は共通の送信、spinner、HTTP status、stream 分岐を処理する。
func RunChatCompletions(ctx context.Context, executor ChatCompletionsExecutor, req *http.Request, options ChatCompletionsRunOptions) (string, error) {
	if executor == nil {
		return "", fmt.Errorf("openai compatible executor is nil")
	}
	if req == nil {
		return "", fmt.Errorf("openai compatible request is nil")
	}
	if options.StreamHandler == nil {
		return "", fmt.Errorf("openai compatible stream handler is nil")
	}

	nonStreamHandler := options.NonStreamHandler
	if nonStreamHandler == nil {
		nonStreamHandler = api.HandleNonStreamingResponse
	}

	spinner := api.StartThinkingSpinner(ctx, options.ImageMode, options.SpinnerSuffix)
	resp, err := executor.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		if options.RequestErrorPrefix != "" {
			return "", fmt.Errorf("%s: %w", options.RequestErrorPrefix, err)
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, executor.Name())
	}

	if options.ForceStreaming || strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return options.StreamHandler(ctx, resp, spinner)
	}
	return nonStreamHandler(ctx, resp, spinner)
}

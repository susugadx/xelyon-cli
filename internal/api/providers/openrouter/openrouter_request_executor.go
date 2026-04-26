package openrouter

import (
	"context"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

// request executor 層: payload を HTTP request として送信し、
// protocol ごとのレスポンス処理へ接続する。

func (p *Provider) newOpenRouterJSONRequest(ctx context.Context, apiURL string, payload []byte) (*http.Request, error) {
	req, err := openaicompat.NewBearerJSONBytesRequest(ctx, apiURL, p.APIKey, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")
	return req, nil
}

func (p *Provider) executeOpenAICompatRequest(ctx context.Context, payload []byte, imageMode bool) (string, error) {
	req, err := p.newOpenRouterJSONRequest(ctx, p.APIURL, payload)
	if err != nil {
		return "", err
	}

	return openaicompat.RunChatCompletions(ctx, p, req, openaicompat.ChatCompletionsRunOptions{
		ImageMode:        imageMode,
		StreamHandler:    p.handleStreamingResponse,
		NonStreamHandler: p.handleNonStreamingResponse,
	})
}

func (p *Provider) executeClaudeStreamingRequest(ctx context.Context, apiURL string, payload []byte, imageMode bool) (string, error) {
	req, err := p.newOpenRouterJSONRequest(ctx, apiURL, payload)
	if err != nil {
		return "", err
	}

	spinner := api.StartThinkingSpinner(ctx, imageMode, "")
	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}
	return p.handleClaudeStreamingResponse(ctx, resp, spinner)
}

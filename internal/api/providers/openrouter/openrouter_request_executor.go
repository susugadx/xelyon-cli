package openrouter

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// request executor 層: payload を HTTP request として送信し、
// protocol ごとのレスポンス処理へ接続する。

func (p *Provider) newOpenRouterJSONRequest(ctx context.Context, apiURL string, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")
	return req, nil
}

func (p *Provider) executeOpenAICompatRequest(ctx context.Context, payload []byte, imageMode bool) (string, error) {
	req, err := p.newOpenRouterJSONRequest(ctx, p.APIURL, payload)
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

	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return p.handleStreamingResponse(ctx, resp, spinner)
	}
	return p.handleNonStreamingResponse(ctx, resp, spinner)
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

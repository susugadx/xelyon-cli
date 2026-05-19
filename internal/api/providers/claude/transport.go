package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type requestResult struct {
	Response *http.Response
	Spinner  *ui.Spinner
}

// executeRequest はClaude API呼び出しの共通処理
// withImage: 画像付きリクエストの場合はtrue（スピナー表示に影響）
func (p *Provider) executeRequest(ctx context.Context, reqBody interface{}, model string, contextManagement *ContextManagement, withImage bool) (*requestResult, error) {
	req, err := p.newAnthropicRequest(ctx, reqBody, model, contextManagement)
	if err != nil {
		return nil, err
	}

	spinner := api.StartThinkingSpinner(ctx, withImage, "")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return nil, err
	}

	if resp.StatusCode != 200 {
		spinner.Stop()
		defer resp.Body.Close()

		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	return &requestResult{Response: resp, Spinner: spinner}, nil
}

func (p *Provider) newAnthropicRequest(ctx context.Context, reqBody interface{}, model string, contextManagement *ContextManagement, betaDefaults ...string) (*http.Request, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	applyAnthropicHeaders(req, p.anthropicHeaders(ctx, model, contextManagement, betaDefaults...))
	return req, nil
}

func (p *Provider) anthropicHeaders(ctx context.Context, model string, contextManagement *ContextManagement, betaDefaults ...string) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("x-api-key", p.APIKey)
	headers.Set("anthropic-version", p.anthropicVersion(ctx, model))

	betaHeaders := p.anthropicBetaHeaders(ctx, model, contextManagement, betaDefaults...)
	if len(betaHeaders) > 0 {
		headers.Set("anthropic-beta", strings.Join(betaHeaders, ","))
	}
	return headers
}

func (p *Provider) anthropicVersion(ctx context.Context, model string) string {
	cfg := config.FromContext(ctx)
	lookupProvider := cfg.RuntimeProviderConfigKey(p.configLookupKey(), model)
	pCfg, _ := cfg.GetProviderModelConfig(lookupProvider)

	if pCfg.AnthropicVersion != "" {
		return pCfg.AnthropicVersion
	}
	return defaultAnthropicVersion
}

func (p *Provider) anthropicBetaHeaders(ctx context.Context, model string, contextManagement *ContextManagement, betaDefaults ...string) []string {
	cfg := config.FromContext(ctx)
	lookupProvider := cfg.RuntimeProviderConfigKey(p.configLookupKey(), model)
	pCfg, _ := cfg.GetProviderModelConfig(lookupProvider)

	if len(betaDefaults) == 0 {
		betaHeaders := make([]string, 0)
		if len(pCfg.AnthropicBeta) > 0 {
			betaHeaders = append(betaHeaders, pCfg.AnthropicBeta...)
		}
		return MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
	}

	seen := make(map[string]bool, len(betaDefaults)+len(pCfg.AnthropicBeta))
	betaHeaders := make([]string, 0, len(betaDefaults)+len(pCfg.AnthropicBeta))
	for _, header := range betaDefaults {
		header = strings.TrimSpace(header)
		if header == "" || seen[header] {
			continue
		}
		seen[header] = true
		betaHeaders = append(betaHeaders, header)
	}
	for _, header := range pCfg.AnthropicBeta {
		header = strings.TrimSpace(header)
		if header == "" || seen[header] {
			continue
		}
		seen[header] = true
		betaHeaders = append(betaHeaders, header)
	}
	return MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
}

func applyAnthropicHeaders(req *http.Request, headers http.Header) {
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

// processResponse はレスポンス処理（ストリーミング/非ストリーミング）
func (p *Provider) processResponse(ctx context.Context, result *requestResult) (string, error) {
	defer result.Response.Body.Close()

	contentType := result.Response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return p.handleStreamingResponse(ctx, result.Response, result.Spinner)
	}
	return p.handleNonStreamingResponse(ctx, result.Response, result.Spinner)
}

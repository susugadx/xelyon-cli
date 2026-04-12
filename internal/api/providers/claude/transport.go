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
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)

	cfg := config.FromContext(ctx)
	lookupProvider := cfg.RuntimeProviderConfigKey(p.configLookupKey(), model)
	pCfg, _ := cfg.GetProviderModelConfig(lookupProvider)

	// Anthropic Version
	version := pCfg.AnthropicVersion
	if version == "" {
		version = "2023-06-01"
	}
	req.Header.Set("anthropic-version", version)

	// Anthropic Beta
	betaHeaders := make([]string, 0)
	if len(pCfg.AnthropicBeta) > 0 {
		betaHeaders = append(betaHeaders, pCfg.AnthropicBeta...)
	}
	betaHeaders = MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
	if len(betaHeaders) > 0 {
		req.Header.Set("anthropic-beta", strings.Join(betaHeaders, ","))
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

// processResponse はレスポンス処理（ストリーミング/非ストリーミング）
func (p *Provider) processResponse(ctx context.Context, result *requestResult) (string, error) {
	defer result.Response.Body.Close()

	contentType := result.Response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return p.handleStreamingResponse(ctx, result.Response, result.Spinner)
	}
	return p.handleNonStreamingResponse(ctx, result.Response, result.Spinner)
}

// isCompactionSupported は Compaction API 対応モデルか判定
// 現時点では Opus 4.6 のみ

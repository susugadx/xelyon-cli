package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// CreateCachedContent は新しいコンテキストキャッシュを作成する
// model: "models/gemini-1.5-pro-001" など
// ttl: "300s" などのDuration文字列 (省略時はAPIデフォルト1時間)
// tools, toolConfig: キャッシュに含めるツール定義（nilの場合は含めない）
func (p *Provider) CreateCachedContent(ctx context.Context, model string, systemPrompt string, contents []api.Message, ttl string, tools []api.GeminiToolConfig, toolConfig *GeminiToolConfigWrapper) (*GeminiCachedContentResponse, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/cachedContents"

	// システムプロンプトの変換
	var sysInst *GeminiSystemInstruction
	if systemPrompt != "" {
		sysInst = &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// モデル名に "models/" プレフィックスを強制
	if !strings.HasPrefix(model, "models/") {
		model = "models/" + model
	}

	reqBody := GeminiCachedContentRequest{
		Model:             model,
		Contents:          geminiFunctionHistoryContents(contents, omitEmptyTextHistoryPart),
		SystemInstruction: sysInst,
		Tools:             tools,
		ToolConfig:        toolConfig,
		TTL:               ttl,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cached content request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.doRequestWithRetry(ctx, req, bodyBytes)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result GeminiCachedContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteCachedContent は指定されたキャッシュを削除する
// name: "cachedContents/123..."
func (p *Provider) DeleteCachedContent(ctx context.Context, name string) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s", name)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

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
func (p *Provider) CreateCachedContent(ctx context.Context, model string, systemPrompt string, contents []api.Message, ttl string) (*GeminiCachedContentResponse, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/cachedContents"

	// システムプロンプトの変換
	var sysInst *GeminiSystemInstruction
	if systemPrompt != "" {
		sysInst = &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// コンテンツの変換（Gemini API は role: "user" / "model" のみ受け付ける）
	geminiContents := make([]interface{}, 0, len(contents))
	for _, msg := range contents {
		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			// assistant の functionCall パート → role: "model"
			parts := make([]interface{}, 0, len(msg.ToolCalls)+1)
			if msg.Content != "" {
				parts = append(parts, GeminiPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				parts = append(parts, GeminiFunctionCallPart{
					FunctionCall: GeminiFunctionCallData{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
			geminiContents = append(geminiContents, GeminiGenericContent{
				Parts: parts,
				Role:  "model",
			})

		case msg.Role == "tool" && msg.ToolCallID != "":
			// functionResponse パート → role: "user"（Gemini API仕様）
			toolName := msg.ToolName
			if toolName == "" {
				toolName = extractToolNameFromContent(msg.Content)
			}
			geminiContents = append(geminiContents, GeminiGenericContent{
				Parts: []interface{}{
					GeminiFunctionResponsePart{
						FunctionResponse: GeminiFunctionResponseData{
							Name: toolName,
							Response: map[string]any{
								"result": msg.Content,
							},
						},
					},
				},
				Role: "user",
			})

		default:
			// 通常のテキストメッセージ: "assistant" → "model", それ以外 → "user"
			role := "user"
			if msg.Role == "assistant" {
				role = "model"
			}
			parts := []GeminiPart{}
			if msg.Content != "" {
				parts = append(parts, GeminiPart{Text: msg.Content})
			}
			geminiContents = append(geminiContents, GeminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	// モデル名に "models/" プレフィックスを強制
	if !strings.HasPrefix(model, "models/") {
		model = "models/" + model
	}

	reqBody := GeminiCachedContentRequest{
		Model:             model,
		Contents:          geminiContents,
		SystemInstruction: sysInst,
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

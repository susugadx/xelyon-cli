package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultCompactURL = "https://api.openai.com/v1/responses/compact"

// CompactRequest は /responses/compact リクエスト
type CompactRequest struct {
	Model        string      `json:"model"`
	Input        []InputItem `json:"input"` // フル会話ウィンドウ
	Instructions string      `json:"instructions,omitempty"`
}

// CompactResponse は /responses/compact レスポンス
type CompactResponse struct {
	Output []InputItem   `json:"output"` // 圧縮済みアイテム（次の /responses に使用）
	Model  string        `json:"model"`
	Usage  *CompactUsage `json:"usage,omitempty"`
}

// CompactUsage はトークン使用量
type CompactUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// SupportsCompact は Compact API 対応を返す
func (p *OpenAIProvider) SupportsCompact() bool {
	return true
}

// CompactHistory は会話履歴を Compact API で圧縮
func (p *OpenAIProvider) CompactHistory(ctx context.Context, input []InputItem, model, instructions string) (*CompactResponse, error) {
	// Compact API URL
	compactURL := os.Getenv("OPENAI_COMPACT_URL")
	if compactURL == "" {
		compactURL = defaultCompactURL
	}

	// モデルが指定されていない場合はデフォルト
	if model == "" {
		model = config.GetGlobalConfig().DefaultModel
	}

	reqBody := CompactRequest{
		Model:        model,
		Input:        input,
		Instructions: instructions,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal compact request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", compactURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create compact request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("compact API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, sanitizeErrorMessage(body, resp.StatusCode)
	}

	var compactResp CompactResponse
	if err := json.NewDecoder(resp.Body).Decode(&compactResp); err != nil {
		return nil, fmt.Errorf("failed to decode compact response: %w", err)
	}

	return &compactResp, nil
}

// ConvertHistoryToInputItems は History を InputItem 形式に変換
// Compact API に送信するためのフル会話ウィンドウを構築
func ConvertHistoryToInputItems(history []Message) []InputItem {
	items := make([]InputItem, 0, len(history))
	for _, msg := range history {
		items = append(items, InputItem{
			Type:    "message",
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return items
}

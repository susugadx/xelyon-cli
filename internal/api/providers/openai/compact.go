package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultCompactURL = "https://api.openai.com/v1/responses/compact"

// CompactRequest は /responses/compact リクエスト
type CompactRequest struct {
	Model        string          `json:"model"`
	Input        []api.InputItem `json:"input"` // フル会話ウィンドウ
	Instructions string          `json:"instructions,omitempty"`
}

// CompactResponse は api.CompactResponse のエイリアス
type CompactResponse = api.CompactResponse

// SupportsCompact は Compact API 対応を返す
func (p *Provider) SupportsCompact() bool {
	return true
}

// CompactHistory は会話履歴を Compact API で圧縮
func (p *Provider) CompactHistory(ctx context.Context, input []api.InputItem, model, instructions string) (*CompactResponse, error) {
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
		return nil, api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	var compactResp CompactResponse
	if err := json.NewDecoder(resp.Body).Decode(&compactResp); err != nil {
		return nil, fmt.Errorf("failed to decode compact response: %w", err)
	}

	return &compactResp, nil
}

// ConvertHistoryToInputItems は api.ConvertHistoryToInputItems のエイリアス
func ConvertHistoryToInputItems(history []api.Message) []api.InputItem {
	return api.ConvertHistoryToInputItems(history)
}

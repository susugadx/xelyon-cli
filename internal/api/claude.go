package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const claudeURL = "https://api.anthropic.com/v1/messages"

// ClaudeProvider はClaude (Anthropic) APIのプロバイダー実装
type ClaudeProvider struct {
	apiKey string
}

// NewClaudeProvider は新しいClaudeProviderを作成
func NewClaudeProvider(apiKey string) *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: apiKey,
	}
}

// Name はプロバイダー名を返す
func (p *ClaudeProvider) Name() string {
	return "Claude"
}

// ClaudeMessage はClaudeのメッセージ構造
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest はClaude APIリクエスト
type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

// ClaudeDelta はストリームの差分
type ClaudeDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeStreamEvent はストリームイベント
type ClaudeStreamEvent struct {
	Type  string      `json:"type"`
	Delta ClaudeDelta `json:"delta"`
}

// ClaudeContent はレスポンスのコンテンツ
type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeResponse は通常レスポンス
type ClaudeResponse struct {
	Content []ClaudeContent `json:"content"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *ClaudeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// モデル名はそのまま使用（ハードコードしない）
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// Claudeのメッセージ構造に変換
	var messages []ClaudeMessage
	for _, msg := range history {
		messages = append(messages, ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := ClaudeRequest{
		Model:     model,
		Messages:  messages,
		System:    systemPrompt,
		MaxTokens: 4096,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", claudeURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Thinking")

	client := &http.Client{
		Timeout: config.DefaultHTTPTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Content-Typeでストリーミング対応を判定
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *ClaudeProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true

	for scanner.Scan() {
		// contextキャンセルチェック
		select {
		case <-ctx.Done():
			spinner.Stop()
			return "", ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			var event ClaudeStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			// content_block_delta イベントのみ処理
			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
				content := event.Delta.Text

				// 最初のコンテンツでスピナー停止
				if firstChunk && content != "" {
					spinner.Stop()
					firstChunk = false
				}

				fmt.Print(content)
				fullResponse.WriteString(content)
			}

			// message_stop イベントで終了
			if event.Type == "message_stop" {
				break
			}
		}
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *ClaudeProvider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Content[0].Text
	fmt.Println(content)
	return content, nil
}

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
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

const openaiURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider はOpenAI APIのプロバイダー実装
type OpenAIProvider struct {
	apiKey string
}

// NewOpenAIProvider は新しいOpenAIProviderを作成
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
	}
}

// Name はプロバイダー名を返す
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *OpenAIProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// メッセージ構築
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名はそのまま使用（ハードコードしない）
	if model == "" {
		model = "gpt-4o"
	}

	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openaiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Thinking")

	client := &http.Client{
		Timeout: 30 * time.Second,
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
func (p *OpenAIProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
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
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content

				// 最初のコンテンツでスピナー停止
				if firstChunk && content != "" {
					spinner.Stop()
					firstChunk = false
				}

				fmt.Print(content)
				fullResponse.WriteString(content)
			}
		}
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *OpenAIProvider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Choices[0].Message.Content
	fmt.Println(content)
	return content, nil
}

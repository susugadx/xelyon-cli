package groq

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("groq", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultGroqURL = "https://api.groq.com/openai/v1/chat/completions"

// Provider はGroq APIのプロバイダー実装（OpenAI互換）
type Provider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("GROQ_API_URL")
	if apiURL == "" {
		apiURL = defaultGroqURL
	}

	return &Provider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "Groq"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return false
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// Extended Thinking 非対応警告
	cfg := config.GetGlobalConfig()
	if cfg.Thinking.Enabled {
		yellow.Println("⚠️  Warning: Groq does not support Extended Thinking. Proceeding without it.")
	}

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名を設定（config優先、フォールバックはllama-3.3-70b-versatile）
	model = api.GetDefaultModel(model, "groq", "llama-3.3-70b-versatile")

	reqBody := api.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(false, "")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
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
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
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

			var streamResp api.StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				// JSONパースエラーを警告（データ損失を防ぐため記録）
				fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to parse streaming response: %v\n", err)
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

	// スキャナーのI/Oエラーチェック
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(resp, spinner)
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応：テキストのみ送信）
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// Groqは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Println("Warning: Groq does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// APIURL はテスト用にAPIURLを公開
func (p *Provider) APIURL() string {
	return p.apiURL
}

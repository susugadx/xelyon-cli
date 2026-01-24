package deepseek

import (
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
)

func init() {
	api.RegisterProvider("deepseek", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultDeepSeekURL = "https://api.deepseek.com/chat/completions"

// Provider はDeepSeek APIのプロバイダー実装
type Provider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("DEEPSEEK_API_URL")
	if apiURL == "" {
		apiURL = defaultDeepSeekURL
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
	return "DeepSeek"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return false
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名を設定（config優先、フォールバックはdeepseek-chat）
	model = api.GetDefaultModel(model, "deepseek", "deepseek-chat")

	// Extended Thinking 有効時は deepseek-reasoner に切り替え
	cfg := config.GetGlobalConfig()
	if cfg.Thinking.Enabled {
		model = "deepseek-reasoner"
	}

	// モデル名マッピング
	actualModel := getActualModel(model)

	reqBody := api.ChatRequest{
		Model:    actualModel,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinnerSuffix := ""
	if cfg.Thinking.Enabled {
		spinnerSuffix = "Reasoner"
	}
	spinner := api.StartThinkingSpinner(false, spinnerSuffix)

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("DeepSeek API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	// ストリーミング処理（共通パーサー使用）
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		// レスポンス構造を検証
		if err := api.ValidateStreamResponse([]byte(data)); err != nil {
			return "", false, fmt.Errorf("invalid response structure: %w", err)
		}

		var streamResp api.StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", false, err
		}

		if len(streamResp.Choices) > 0 {
			return streamResp.Choices[0].Delta.Content, false, nil
		}

		return "", false, nil
	}

	return api.ParseStreamingResponse(ctx, resp, spinner, parser)
}

// getActualModel はモデル名を実際のAPI用に変換
func getActualModel(model string) string {
	switch model {
	case "deepseek-chat", "":
		return "deepseek-chat"
	case "deepseek-coder":
		return "deepseek-coder"
	case "deepseek-reasoner":
		return "deepseek-reasoner"
	default:
		return "deepseek-chat"
	}
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応：テキストのみ送信）
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// DeepSeekは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Println("Warning: DeepSeek does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// APIURL はテスト用にAPIURLを公開
func (p *Provider) APIURL() string {
	return p.apiURL
}

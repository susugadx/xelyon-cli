package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultDeepSeekURL = "https://api.deepseek.com/chat/completions"

// Message はチャットメッセージ（provider.goで定義されているが、ここでも使用）
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekProvider はDeepSeek APIのプロバイダー実装
type DeepSeekProvider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// NewDeepSeekProvider は新しいDeepSeekProviderを作成
func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("DEEPSEEK_API_URL")
	if apiURL == "" {
		apiURL = defaultDeepSeekURL
	}

	return &DeepSeekProvider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *DeepSeekProvider) Name() string {
	return "DeepSeek"
}

// ChatRequest はAPIリクエスト
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Delta はストリームレスポンスの差分
type Delta struct {
	Content string `json:"content"`
}

// StreamChoice はストリームの選択肢
type StreamChoice struct {
	Delta Delta `json:"delta"`
}

// StreamResponse はストリームレスポンス
type StreamResponse struct {
	Choices []StreamChoice `json:"choices"`
}

// Choice は通常レスポンスの選択肢
type Choice struct {
	Message Message `json:"message"`
}

// ChatResponse は通常レスポンス
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *DeepSeekProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// メッセージ構築
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名マッピング
	actualModel := getActualModel(model)

	reqBody := ChatRequest{
		Model:    actualModel,
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
	spinner := ui.NewSpinner()
	spinner.Start("Thinking")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()

		// レート制限チェック
		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return "", sanitizeErrorMessage(body, resp.StatusCode)
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
		if err := ValidateStreamResponse([]byte(data)); err != nil {
			return "", false, fmt.Errorf("invalid response structure: %w", err)
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", false, err
		}

		if len(streamResp.Choices) > 0 {
			return streamResp.Choices[0].Delta.Content, false, nil
		}

		return "", false, nil
	}

	return ParseStreamingResponse(ctx, resp, spinner, parser)
}

// ChatWithTools はツール対応の会話を行う（ストリーミング）
// Deprecated: 後方互換性のために残されています。NewDeepSeekProvider + Client を使用してください。
func ChatWithTools(systemPrompt string, history []Message, model string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	// 新しいProvider経由で実行
	provider := NewDeepSeekProvider(apiKey)
	client := NewClient(provider)
	return client.ChatWithTools(systemPrompt, history, model)
}

// AskDeepSeekStream は従来のストリーミング質問（後方互換）
// Deprecated: 後方互換性のために残されています。ChatWithTools を使用してください。
func AskDeepSeekStream(query string, context string, model string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	systemPrompt := "You are an excellent programming assistant. Answer based on the provided context. When modifying code, always present the complete code wrapped in triple backticks."

	userContent := query
	if context != "" {
		userContent = fmt.Sprintf("## Context:\n%s\n\n## Question:\n%s", context, query)
	}

	// 新しいProvider経由で実行
	provider := NewDeepSeekProvider(apiKey)
	client := NewClient(provider)

	history := []Message{
		{Role: "user", Content: userContent},
	}

	return client.ChatWithTools(systemPrompt, history, model)
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

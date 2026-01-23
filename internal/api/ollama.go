package api

import (
	"bufio"
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

// OllamaProvider はOllama APIのプロバイダー実装
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewOllamaProvider は新しいOllamaProviderを作成
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *OllamaProvider) Name() string {
	return "Ollama"
}

// SupportsImages は画像入力対応を返す
func (p *OllamaProvider) SupportsImages() bool {
	return false
}

// OllamaRequest はOllama APIリクエスト
type OllamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// OllamaMessageContent はOllamaのメッセージコンテンツ
type OllamaMessageContent struct {
	Content string `json:"content"`
}

// OllamaStreamResponse はOllamaのストリームレスポンス
type OllamaStreamResponse struct {
	Message OllamaMessageContent `json:"message"`
	Done    bool                 `json:"done"`
}

// OllamaModel はモデル情報
type OllamaModel struct {
	Name string `json:"name"`
}

// OllamaTagsResponse はモデル一覧のレスポンス
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *OllamaProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// Extended Thinking 注意メッセージ（モデル依存）
	cfg := config.GetGlobalConfig()
	if cfg.Thinking.Enabled {
		yellow.Println("⚠️  Note: Extended Thinking depends on your model (use R1/QwQ for best results).")
	}

	// モデル名を設定（config優先、フォールバックはllama3）
	model = GetDefaultModel(model, "ollama", "llama3")

	// メッセージ構築
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := OllamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := p.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	// スピナー開始
	spinner := StartThinkingSpinner(false, "")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		// 接続エラー時は親切なメッセージを表示
		if strings.Contains(err.Error(), "connection refused") {
			return "", fmt.Errorf("ollama is not running. Please start it with `ollama serve`")
		}
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

	// Ollamaは常にJSONLストリーム形式
	return p.handleStreamingResponse(ctx, resp, spinner)
}

// handleStreamingResponse はストリーミングレスポンスを処理（JSON Lines形式）
func (p *OllamaProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
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

		var streamResp OllamaStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			// JSONパースエラーを警告（データ損失を防ぐため記録）
			fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to parse streaming response: %v\n", err)
			continue
		}

		// done=trueで終了
		if streamResp.Done {
			break
		}

		content := streamResp.Message.Content
		if content != "" {
			// 最初のコンテンツでスピナー停止
			if firstChunk {
				spinner.Stop()
				firstChunk = false
			}

			fmt.Print(content)
			fullResponse.WriteString(content)
		}
	}

	// スキャナーのI/Oエラーチェック
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// ListModels はインストール済みモデルを取得
func (p *OllamaProvider) ListModels() ([]string, error) {
	url := p.baseURL + "/api/tags"
	resp, err := p.httpClient.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("ollama is not running. Please start it with `ollama serve`")
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// レート制限チェック
		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, sanitizeErrorMessage(body, resp.StatusCode)
	}

	var result OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応：テキストのみ送信）
func (p *OllamaProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	// Ollamaは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Println("Warning: Ollama does not support image input. The image will be ignored.")
	}
	history = append(history, Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

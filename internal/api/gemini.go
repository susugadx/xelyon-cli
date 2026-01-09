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

// GeminiProvider はGemini APIのプロバイダー実装
type GeminiProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGeminiProvider は新しいGeminiProviderを作成
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *GeminiProvider) Name() string {
	return "Gemini"
}

// GeminiPart はGeminiの parts 構造
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiContent はGeminiの contents 構造
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"` // "user" or "model"
}

// GeminiRequest はGemini APIリクエスト
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiCandidate はレスポンスの候補
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiResponse はGeminiレスポンス
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *GeminiProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// モデル名はそのまま使用（ハードコードしない）
	if model == "" {
		model = "gemini-2.0-flash-exp"
	}

	// Geminiのメッセージ構造に変換
	var contents []GeminiContent

	// System promptを最初のユーザーメッセージとして追加
	if systemPrompt != "" {
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
		// ダミーのモデル応答を追加（Geminiは交互のroleを期待）
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: "Understood. I'll follow these instructions."}},
			Role:  "model",
		})
	}

	// 会話履歴を変換
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: msg.Content}},
			Role:  role,
		})
	}

	reqBody := GeminiRequest{
		Contents: contents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API endpoint（ストリーミング）
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent", model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey) // APIキーはヘッダーで送信（セキュリティ向上）

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

	// Content-Typeでストリーミング対応を判定
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "application/json")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *GeminiProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
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

		// Geminiはdata:プレフィックスなしでJSONを返す場合がある
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
		}

		if line == "[DONE]" {
			break
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal([]byte(line), &geminiResp); err != nil {
			continue
		}

		if len(geminiResp.Candidates) > 0 {
			candidate := geminiResp.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				content := candidate.Content.Parts[0].Text

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
func (p *GeminiProvider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Candidates[0].Content.Parts[0].Text
	fmt.Println(content)
	return content, nil
}

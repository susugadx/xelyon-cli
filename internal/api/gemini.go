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

const defaultGeminiURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent"

// getGeminiURL は環境変数またはデフォルトのURLテンプレートを使用してURLを生成
func getGeminiURL(model string) string {
	if baseURL := os.Getenv("GEMINI_API_URL"); baseURL != "" {
		// 環境変数が設定されている場合はそのまま使用（テスト用）
		return baseURL
	}
	return fmt.Sprintf(defaultGeminiURLTemplate, model)
}

// GeminiProvider はGemini APIのプロバイダー実装
type GeminiProvider struct {
	apiKey     string
	httpClient *http.Client
	mcpEnabled bool // MCP有効時はテキストモードにフォールバック
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

// SetMCPEnabled はMCPが有効かどうかを設定する
// MCP有効時はFunction Callingではなくテキストモードにフォールバック
func (p *GeminiProvider) SetMCPEnabled(enabled bool) {
	p.mcpEnabled = enabled
}

// SupportsImages は画像入力対応を返す
func (p *GeminiProvider) SupportsImages() bool {
	return true
}

// GeminiPart はGeminiの parts 構造（テキストのみ）
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiMultimodalPart はマルチモーダル対応のparts構造
type GeminiMultimodalPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

// GeminiInlineData は画像データ
type GeminiInlineData struct {
	MimeType string `json:"mime_type"` // "image/png", "image/jpeg" etc
	Data     string `json:"data"`      // Base64エンコードされたデータ
}

// GeminiMultimodalContent はマルチモーダル対応のcontents構造
type GeminiMultimodalContent struct {
	Parts []GeminiMultimodalPart `json:"parts"`
	Role  string                 `json:"role,omitempty"` // "user" or "model"
}

// GeminiMultimodalRequest はマルチモーダルAPIリクエスト
type GeminiMultimodalRequest struct {
	Contents []interface{} `json:"contents"` // GeminiContent or GeminiMultimodalContent
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

// ===== Function Calling API structures =====

// GeminiFunctionPart はtext または functionCall を含むパート
type GeminiFunctionPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`
}

// GeminiFunctionContent はFunction Calling対応のコンテンツ
type GeminiFunctionContent struct {
	Parts []GeminiFunctionPart `json:"parts"`
	Role  string               `json:"role,omitempty"`
}

// GeminiFunctionCandidate はFunction Calling対応の候補
type GeminiFunctionCandidate struct {
	Content GeminiFunctionContent `json:"content"`
}

// GeminiFunctionResponse はFunction Calling対応のレスポンス
type GeminiFunctionResponse struct {
	Candidates []GeminiFunctionCandidate `json:"candidates"`
}

// GeminiRequestWithTools はtools を含むリクエスト
type GeminiRequestWithTools struct {
	Contents []interface{}      `json:"contents"`
	Tools    []GeminiToolConfig `json:"tools,omitempty"`
}

// ChatWithTools は Provider interface の実装（context対応）
// MCP有効時またはGEMINI_FUNCTION_CALLING=0の場合はテキストモードを使用
func (p *GeminiProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// デバッグモード
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	// MCP有効時はテキストモードにフォールバック
	if p.mcpEnabled {
		if debug {
			fmt.Println("[DEBUG Gemini] Mode: TextMode (MCP enabled)")
		}
		return p.chatWithTextMode(ctx, systemPrompt, history, model)
	}

	// 環境変数でFunction Callingを制御（デフォルト: 有効）
	useFunctionCalling := os.Getenv("GEMINI_FUNCTION_CALLING") != "0"

	if debug {
		fmt.Printf("[DEBUG Gemini] mcpEnabled=%v, GEMINI_FUNCTION_CALLING=%q, useFunctionCalling=%v\n",
			p.mcpEnabled, os.Getenv("GEMINI_FUNCTION_CALLING"), useFunctionCalling)
	}

	if useFunctionCalling {
		if debug {
			fmt.Println("[DEBUG Gemini] Mode: Function Calling")
		}
		result, err := p.chatWithFunctionCalling(ctx, systemPrompt, history, model)
		if err != nil {
			// Function Calling失敗時はテキストモードにフォールバック
			fmt.Printf("Warning: Function Calling failed, falling back to text mode: %v\n", err)
			return p.chatWithTextMode(ctx, systemPrompt, history, model)
		}
		return result, nil
	}

	if debug {
		fmt.Println("[DEBUG Gemini] Mode: TextMode (GEMINI_FUNCTION_CALLING=0)")
	}
	return p.chatWithTextMode(ctx, systemPrompt, history, model)
}

// chatWithTextMode はテキストベースのツール呼び出しモード（従来の実装）
func (p *GeminiProvider) chatWithTextMode(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
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
	url := getGeminiURL(model)

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
			return "", fmt.Errorf("Gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("Gemini API error (status %d): empty response body. Check API key and model name '%s'", resp.StatusCode, model)
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

// getGeminiFunctionCallingURL は Function Calling 用の URL を生成（非ストリーミング）
func getGeminiFunctionCallingURL(model string) string {
	if baseURL := os.Getenv("GEMINI_API_URL"); baseURL != "" {
		return baseURL
	}
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
}

// chatWithFunctionCalling は Function Calling API を使用してツールを呼び出す
func (p *GeminiProvider) chatWithFunctionCalling(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	if model == "" {
		model = "gemini-2.0-flash-exp"
	}

	// メッセージを interface{} スライスに変換（Function Calling リクエスト用）
	var contents []interface{}

	// System prompt を最初のユーザーメッセージとして追加
	if systemPrompt != "" {
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
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

	// Function Calling 用リクエストを構築
	reqBody := GeminiRequestWithTools{
		Contents: contents,
		Tools:    GetGeminiToolDefinitions(),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Function Calling エンドポイント（非ストリーミング）
	url := getGeminiFunctionCallingURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Thinking (Function Calling)")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込み
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		spinner.Stop()
		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		if len(body) == 0 {
			return "", fmt.Errorf("Gemini API error (status %d): empty response body", resp.StatusCode)
		}
		return "", sanitizeErrorMessage(body, resp.StatusCode)
	}

	// Function Calling レスポンスを処理
	return p.handleFunctionCallingResponse(body, spinner)
}

// handleFunctionCallingResponse は Function Calling レスポンスを処理
func (p *GeminiProvider) handleFunctionCallingResponse(body []byte, spinner *ui.Spinner) (string, error) {
	var responses []GeminiFunctionResponse

	// まず配列としてパースを試みる（ストリーミングレスポンス）
	if err := json.Unmarshal(body, &responses); err != nil {
		// 配列でない場合は単一オブジェクトとして試す
		var singleResponse GeminiFunctionResponse
		if err := json.Unmarshal(body, &singleResponse); err != nil {
			if spinner != nil {
				spinner.Stop()
			}
			return "", fmt.Errorf("failed to parse Function Calling response: %w", err)
		}
		responses = []GeminiFunctionResponse{singleResponse}
	}

	if spinner != nil {
		spinner.Stop()
	}

	var fullResponse strings.Builder

	for _, response := range responses {
		if len(response.Candidates) == 0 {
			continue
		}

		candidate := response.Candidates[0]

		for _, part := range candidate.Content.Parts {
			// テキストパートを処理
			if part.Text != "" {
				fmt.Print(part.Text)
				fullResponse.WriteString(part.Text)
			}

			// Function Call パートを処理
			if part.FunctionCall != nil {
				toolJSON := convertFunctionCallToToolJSON(part.FunctionCall)
				fmt.Printf("\n%s", toolJSON)
				fullResponse.WriteString(toolJSON)
			}
		}
	}

	if fullResponse.Len() == 0 {
		return "", fmt.Errorf("no content in Function Calling response")
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *GeminiProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var fullResponse strings.Builder

	// 全レスポンスを読み込み
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// JSON配列としてパース
	var responses []GeminiResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		spinner.Stop()

		// 配列でない場合は単一オブジェクトとして試す
		var singleResp GeminiResponse
		if err := json.Unmarshal(body, &singleResp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		responses = []GeminiResponse{singleResp}
	}

	spinner.Stop()

	// 各レスポンスからテキストを抽出
	firstChunk := true
	for _, geminiResp := range responses {
		if len(geminiResp.Candidates) > 0 {
			candidate := geminiResp.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				content := candidate.Content.Parts[0].Text

				if firstChunk && content != "" {
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

// ChatWithImage は画像付きメッセージで会話を行う
func (p *GeminiProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名の設定
	if model == "" {
		model = "gemini-2.0-flash-exp"
	}

	// contentsを構築
	var contents []interface{}

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

	// 会話履歴を変換（テキストのみ）
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

	// 画像付きユーザーメッセージを追加
	multimodalContent := GeminiMultimodalContent{
		Role: "user",
		Parts: []GeminiMultimodalPart{
			{
				InlineData: &GeminiInlineData{
					MimeType: image.MediaType,
					Data:     image.Base64,
				},
			},
			{
				Text: userMessage,
			},
		},
	}
	contents = append(contents, multimodalContent)

	reqBody := GeminiMultimodalRequest{
		Contents: contents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API endpoint（ストリーミング）
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Analyzing image")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()

		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("Gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("Gemini API error (status %d): empty response body. Check API key and model name '%s'", resp.StatusCode, model)
		}
		return "", sanitizeErrorMessage(body, resp.StatusCode)
	}

	// ストリーミング処理
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "application/json")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

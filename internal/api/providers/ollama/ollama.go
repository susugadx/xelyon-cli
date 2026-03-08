package ollama

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

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("ollama", func(baseURL string) (api.Provider, error) {
		return New(baseURL), nil
	})
}

var yellow = color.New(color.FgYellow)

// toolCallAccumulator はストリーミング中のtool_callを蓄積する
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// Provider はOllama APIのプロバイダー実装
type Provider struct {
	baseURL       string
	httpClient    *http.Client
	mcpTools      []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback api.UsageCallback    // トークン使用量コールバック
	toolChoice    *string              // tool_choice 強制用
}

// New は新しいProviderを作成
func New(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Provider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "Ollama"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return false
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
// OLLAMA_FUNCTION_CALLING=0 で無効化可能
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("OLLAMA_FUNCTION_CALLING") != "0"
}

// OllamaOptions はOllamaの生成オプション
type OllamaOptions struct {
	NumPredict int `json:"num_predict,omitempty"` // 最大出力トークン数
}

// OllamaRequest はOllama APIリクエスト
type OllamaRequest struct {
	Model      string           `json:"model"`
	Messages   []api.Message    `json:"messages"`
	Stream     bool             `json:"stream"`
	Options    *OllamaOptions   `json:"options,omitempty"`     // 生成オプション
	Tools      []api.OpenAITool `json:"tools,omitempty"`       // Function Calling用
	ToolChoice string           `json:"tool_choice,omitempty"` // "auto", "none"
}

// OllamaMessageContent はOllamaのメッセージコンテンツ
type OllamaMessageContent struct {
	Content   string               `json:"content"`
	ToolCalls []api.OpenAIToolCall `json:"tool_calls,omitempty"` // Function Calling用
}

// OllamaStreamResponse はOllamaのストリームレスポンス
type OllamaStreamResponse struct {
	Message         OllamaMessageContent `json:"message"`
	Done            bool                 `json:"done"`
	PromptEvalCount int                  `json:"prompt_eval_count,omitempty"` // 入力トークン数（done=true時）
	EvalCount       int                  `json:"eval_count,omitempty"`        // 出力トークン数（done=true時）
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
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// Extended Thinking 注意メッセージ（モデル依存）
	if api.IsThinkingEnabled(ctx) {
		yellow.Println("⚠️  Note: Extended Thinking depends on your model (use R1/QwQ for best results).")
	}

	// モデル名を設定（config優先、フォールバックはllama3）
	model = api.GetDefaultModelWithContext(ctx, model, "ollama", "llama3")

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := OllamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options: &OllamaOptions{
			NumPredict: api.GetMaxOutputTokens(ctx, "ollama", model),
		},
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("OLLAMA_FUNCTION_CALLING") != "0" {
		reqBody.Tools = openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools)
		reqBody.ToolChoice = "auto"

		// tool_choice 強制設定がある場合
		if p.toolChoice != nil {
			reqBody.ToolChoice = *p.toolChoice
		}
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
	spinner := api.StartThinkingSpinner(ctx, false, "")

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
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	// Ollamaは常にJSONLストリーム形式
	return p.handleStreamingResponse(ctx, resp, spinner)
}

// handleStreamingResponse はストリーミングレスポンスを処理（JSON Lines形式）
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	var fullResponse strings.Builder
	var toolCallsOutput strings.Builder
	toolCalls := make(map[int]*toolCallAccumulator)
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
			fmt.Fprintf(errOut, "⚠️  Warning: failed to parse streaming response: %v\n", err)
			continue
		}

		// Function Calling: tool_calls を蓄積
		for idx, tc := range streamResp.Message.ToolCalls {
			acc, exists := toolCalls[idx]
			if !exists {
				acc = &toolCallAccumulator{}
				toolCalls[idx] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Function.Name != "" {
				acc.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				// スピナーを再表示
				if !spinner.IsActive() {
					spinner.Start(ui.SpinnerMessageForTool(acc.Name))
				}
				acc.Arguments.Reset() // Ollama は累積ではなく完全な引数を送る
				acc.Arguments.WriteString(tc.Function.Arguments)
			}
		}

		// done=trueで終了
		if streamResp.Done {
			// Usage callback（Ollamaはdone=true時にトークン数を返す）
			if p.usageCallback != nil && (streamResp.PromptEvalCount > 0 || streamResp.EvalCount > 0) {
				p.usageCallback(api.Usage{
					InputTokens:  streamResp.PromptEvalCount,
					OutputTokens: streamResp.EvalCount,
				})
			}

			// tool_calls がある場合は変換
			if len(toolCalls) > 0 {
				for i := 0; i < len(toolCalls); i++ {
					acc := toolCalls[i]
					if acc == nil {
						continue
					}
					tc := &api.OpenAIToolCall{
						ID:   acc.ID,
						Type: "function",
						Function: api.OpenAIToolCallFunction{
							Name:      acc.Name,
							Arguments: acc.Arguments.String(),
						},
					}
					if toolJSON, err := openai.ConvertToolCallToToolJSON(tc); err == nil {
						toolCallsOutput.WriteString(toolJSON)
					}
				}
			}
			break
		}

		content := streamResp.Message.Content
		if content != "" {
			// 最初のコンテンツでスピナー停止 + AI発言ヘッダー表示
			if firstChunk {
				spinner.Stop()
				firstChunk = false
				api.PrintAIHeaderWithContext(ctx)
			}

			_, _ = fmt.Fprint(out, content)
			fullResponse.WriteString(content)
		}
	}

	// スキャナーのI/Oエラーチェック
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	// tool_calls がある場合はそれを返す
	if toolCallsOutput.Len() > 0 {
		spinner.Stop()
		if fullResponse.Len() > 0 {
			_, _ = fmt.Fprintln(out)
			return fullResponse.String() + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}

	_, _ = fmt.Fprintln(out)
	return fullResponse.String(), nil
}

// ListModels はインストール済みモデルを取得
func (p *Provider) ListModels() ([]string, error) {
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
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, api.SanitizeErrorMessage(body, resp.StatusCode)
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
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// Ollamaは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Println("Warning: Ollama does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// BaseURL はテスト用にbaseURLを公開
func (p *Provider) BaseURL() string {
	return p.baseURL
}

// SetMCPTools は MCP ツール定義を設定する（Function Calling用）
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は tool_choice を設定する
func (p *Provider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする
func (p *Provider) ClearToolChoice() {
	p.toolChoice = nil
}

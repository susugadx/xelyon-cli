package openrouter

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
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("openrouter", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultOpenRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// toolCallAccumulator はストリーミング中のtool_callを蓄積する
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// ContentPart はマルチモーダルコンテンツのパート（OpenAI互換）
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // type="text"の場合
	ImageURL *ImageURL `json:"image_url,omitempty"` // type="image_url"の場合
}

// ImageURL は画像URL
type ImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..." 形式
}

// MultimodalMessage はマルチモーダルメッセージ（OpenAI互換）
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// Provider はOpenRouter APIのプロバイダー実装（OpenAI互換 + Claude Compaction対応）
type Provider struct {
	api.BaseProvider
	mcpTools          []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback     api.UsageCallback    // トークン使用量コールバック
	compactionEnabled bool                 // Compaction が有効か
	compactionTrigger int                  // Compaction を開始する入力トークン閾値
	toolChoice        *string              // tool_choice 強制用
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	globalCfg := config.GetGlobalConfig()
	return &Provider{
		BaseProvider:      api.NewBaseProvider("OpenRouter", apiKey, defaultOpenRouterURL, "OPENROUTER_API_URL"),
		compactionEnabled: globalCfg.Compression.ClaudeCompaction,
		compactionTrigger: globalCfg.Compression.CompactionTrigger,
	}
}

// isClaudeModel はモデル名が Claude モデルかを判定
func isClaudeModel(model string) bool {
	return strings.HasPrefix(model, "anthropic/claude-")
}

// isCompactionSupported はモデルが Compaction 対応か判定
func isCompactionSupported(model string) bool {
	return strings.Contains(model, "opus-4-6") || strings.Contains(model, "opus-4-5") ||
		strings.Contains(model, "opus-4.6") || strings.Contains(model, "opus-4.5") ||
		strings.Contains(model, "sonnet-4-6") || strings.Contains(model, "sonnet-4.6")
}

// getAnthropicSkinURL は OpenAI 互換 URL から Anthropic Skin URL を導出
func getAnthropicSkinURL(openaiURL string) string {
	if idx := strings.Index(openaiURL, "/v1/chat/completions"); idx >= 0 {
		return openaiURL[:idx] + "/v1/messages"
	}
	return strings.TrimSuffix(openaiURL, "/chat/completions") + "/messages"
}

// SupportsClaudeCompaction はこのプロバイダーが Claude Compaction に対応しているかを返す
func (p *Provider) SupportsClaudeCompaction() bool {
	model := api.GetDefaultModel("", "openrouter", "anthropic/claude-sonnet-4.6")
	return p.compactionEnabled && isClaudeModel(model) && isCompactionSupported(model)
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("OPENROUTER_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Provider interface の実装
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.GetDefaultModel(model, "openrouter", "anthropic/claude-sonnet-4.6")

	// Claude モデル + Compaction 有効時は Anthropic Skin エンドポイントを使用
	if isClaudeModel(model) && p.compactionEnabled && isCompactionSupported(model) {
		return p.chatWithClaudeAPI(ctx, systemPrompt, history, model, nil)
	}

	if api.IsThinkingEnabled(ctx) {
		yellow.Println("⚠️  Warning: OpenRouter does not support Extended Thinking. Proceeding without it.")
	}

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := api.ChatRequest{
		Model:         model,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens(ctx, "openrouter", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
	}

	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = openai.GetCombinedOpenAITools(p.mcpTools)
		reqBody.ToolChoice = "auto"

		// tool_choice 強制設定がある場合
		if p.toolChoice != nil {
			reqBody.ToolChoice = map[string]interface{}{
				"type": "function",
				"function": map[string]string{
					"name": *p.toolChoice,
				},
			}
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")

	spinner := api.StartThinkingSpinner(ctx, false, "")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

// ChatWithImage は画像付きチャットの実装
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	model = api.GetDefaultModel(model, "openrouter", "anthropic/claude-sonnet-4.6")

	if isClaudeModel(model) && p.compactionEnabled && isCompactionSupported(model) {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.chatWithClaudeAPI(ctx, systemPrompt, history, model, image)
	}

	return p.chatWithImageRequest(ctx, systemPrompt, history, userMessage, image, model)
}

// chatWithImageRequest はOpenAI互換形式での画像送信
func (p *Provider) chatWithImageRequest(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	imageUrl := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)

	content := []ContentPart{
		{Type: "text", Text: userMessage},
		{Type: "image_url", ImageURL: &ImageURL{URL: imageUrl}},
	}

	// マルチモーダルメッセージ構築
	var messages []interface{}
	messages = append(messages, api.Message{Role: "system", Content: systemPrompt})
	for _, msg := range history {
		messages = append(messages, msg)
	}
	messages = append(messages, MultimodalMessage{
		Role:    "user",
		Content: content,
	})

	reqBody := struct {
		Model         string             `json:"model"`
		Messages      []interface{}      `json:"messages"`
		MaxTokens     int                `json:"max_tokens"`
		Stream        bool               `json:"stream"`
		StreamOptions *api.StreamOptions `json:"stream_options,omitempty"`
	}{
		Model:         model,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens(ctx, "openrouter", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")

	spinner := api.StartThinkingSpinner(ctx, true, "")

	resp, err := p.ExecuteRequest(req)
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

// handleStreamingResponse は OpenAI 互換ストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var fullResponse strings.Builder
	var toolCallsOutput strings.Builder
	toolCalls := make(map[int]*toolCallAccumulator)
	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true
	var lastUsage *api.Usage

	for scanner.Scan() {
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
				continue
			}

			if streamResp.Usage != nil {
				cachedTokens := 0
				if streamResp.Usage.PromptTokensDetails != nil {
					cachedTokens = streamResp.Usage.PromptTokensDetails.CachedTokens
				}
				lastUsage = &api.Usage{
					InputTokens:       streamResp.Usage.PromptTokens,
					OutputTokens:      streamResp.Usage.CompletionTokens,
					CachedInputTokens: cachedTokens,
				}
			}

			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]

				if len(choice.Delta.ToolCalls) > 0 {
					for _, tc := range choice.Delta.ToolCalls {
						index := tc.Index
						if _, ok := toolCalls[index]; !ok {
							toolCalls[index] = &toolCallAccumulator{ID: tc.ID, Name: tc.Function.Name}
						}
						if tc.Function.Arguments != "" {
							// スピナーを再表示
							if !spinner.IsActive() {
								spinner.Start(ui.SpinnerMessageForTool(toolCalls[index].Name))
							}
							toolCalls[index].Arguments.WriteString(tc.Function.Arguments)
						}
					}
					continue
				}

				if choice.Delta.Content != "" {
					if firstChunk {
						spinner.Stop()
						firstChunk = false
						api.PrintAIHeader()
					}
					fmt.Print(choice.Delta.Content)
					fullResponse.WriteString(choice.Delta.Content)
				}
			}
		}
	}

	spinner.Stop()

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if p.usageCallback != nil && lastUsage != nil {
		p.usageCallback(*lastUsage)
	}

	for i := 0; i < len(toolCalls); i++ {
		if tc, ok := toolCalls[i]; ok {
			openaiTC := &api.OpenAIToolCall{
				ID: tc.ID,
				Function: api.OpenAIToolCallFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments.String(),
				},
			}
			if toolJSON, err := openai.ConvertToolCallToToolJSON(openaiTC); err == nil {
				toolCallsOutput.WriteString(toolJSON)
			}
		}
	}

	content := fullResponse.String()
	if toolCallsOutput.Len() > 0 {
		if content != "" {
			fmt.Println()
			return content + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}

	fmt.Println()
	return content, nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var apiResp struct {
		Choices []api.Choice        `json:"choices"`
		Usage   api.StreamUsageInfo `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		spinner.Stop()
		return "", err
	}
	spinner.Stop()

	if p.usageCallback != nil {
		cachedTokens := 0
		if apiResp.Usage.PromptTokensDetails != nil {
			cachedTokens = apiResp.Usage.PromptTokensDetails.CachedTokens
		}
		p.usageCallback(api.Usage{
			InputTokens:       apiResp.Usage.PromptTokens,
			OutputTokens:      apiResp.Usage.CompletionTokens,
			CachedInputTokens: cachedTokens,
		})
	}

	if len(apiResp.Choices) > 0 {
		api.PrintAIHeader()
		content := apiResp.Choices[0].Message.Content
		fmt.Println(content)
		return content, nil
	}
	return "", nil
}

// SetMCPTools はツール定義を設定
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback はトークン使用量コールバックを設定
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

// chatWithClaudeAPI は Claude モデル用に Anthropic Skin エンドポイントを使用して通信する
func (p *Provider) chatWithClaudeAPI(ctx context.Context, systemPrompt string, history []api.Message, model string, image *api.ImageData) (string, error) {
	// メッセージを Anthropic 形式に変換
	anthropicMessages := claude.ConvertToAnthropicMessages(history)

	// プロンプトキャッシュ: 最後の2つの user メッセージにブレークポイント設定
	cfg := config.GetGlobalConfig()
	if cfg != nil && cfg.PromptCache.Enabled {
		claude.SetCacheOnLastTwoUserMessages(anthropicMessages)
	}

	// リクエスト構造体（Anthropic Messages API 形式）
	type claudeRequest struct {
		Model             string                    `json:"model"`
		AnthropicVersion  string                    `json:"anthropic_version"`
		AnthropicBeta     []string                  `json:"anthropic_beta,omitempty"`
		MaxTokens         int                       `json:"max_tokens"`
		System            interface{}               `json:"system,omitempty"`
		Messages          []claude.AnthropicMessage `json:"messages"`
		Tools             []claude.ClaudeTool       `json:"tools,omitempty"`
		Stream            bool                      `json:"stream"`
		ContextManagement *claude.ContextManagement `json:"context_management,omitempty"`
	}

	reqBody := claudeRequest{
		Model:            model,
		AnthropicVersion: "2023-06-01",
		MaxTokens:        api.GetMaxOutputTokens(ctx, "openrouter", model),
		System:           api.BuildSystemField(systemPrompt),
		Messages:         anthropicMessages,
		Stream:           true,
	}

	// Tool Use 設定
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = claude.GetCombinedClaudeTools(p.mcpTools)
	}

	// Compaction 設定
	reqBody.ContextManagement = &claude.ContextManagement{
		Edits: []claude.ContextEdit{
			{
				Type: "compact_20260112",
				Trigger: &claude.CompactTrigger{
					Type:  "input_tokens",
					Value: p.compactionTrigger,
				},
			},
		},
	}
	reqBody.AnthropicBeta = []string{"compact-2026-01-12"}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Anthropic Skin URL を導出
	apiURL := getAnthropicSkinURL(p.APIURL)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	// ヘッダー設定（Anthropic Skin 用）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")

	spinner := api.StartThinkingSpinner(ctx, image != nil, "")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	return p.handleClaudeStreamingResponse(ctx, resp, spinner)
}

// handleClaudeStreamingResponse は Anthropic SSE ストリーミングレスポンスを処理する
func (p *Provider) handleClaudeStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var fullResponse strings.Builder
	var toolCallsOutput strings.Builder
	var compactionOutput strings.Builder
	toolUses := make(map[int]*struct {
		ID    string
		Name  string
		Input strings.Builder
	})
	compactionBlocks := make(map[int]*strings.Builder)
	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true
	var lastUsage *api.Usage

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			spinner.Stop()
			content := fullResponse.String()
			if compactionOutput.Len() > 0 {
				content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
			}
			if toolCallsOutput.Len() > 0 {
				if content != "" {
					return content + toolCallsOutput.String(), ctx.Err()
				}
				return toolCallsOutput.String(), ctx.Err()
			}
			return content, ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event claude.StreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			var msgStart struct {
				Message struct {
					Usage claude.StreamUsage `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &msgStart); err == nil {
				usage := msgStart.Message.Usage
				lastUsage = &api.Usage{
					InputTokens:         usage.InputTokens,
					CachedInputTokens:   usage.CacheReadInputTokens,
					CacheCreationTokens: usage.CacheCreationInputTokens,
				}
			}

		case "message_delta":
			var msgDelta struct {
				Usage claude.StreamUsage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &msgDelta); err == nil && msgDelta.Usage.OutputTokens > 0 {
				if lastUsage == nil {
					lastUsage = &api.Usage{}
				}
				lastUsage.OutputTokens = msgDelta.Usage.OutputTokens
			}

		case "message_stop":
			goto done

		case "content_block_start":
			if event.ContentBlock == nil {
				continue
			}
			switch event.ContentBlock.Type {
			case "tool_use":
				toolUses[event.Index] = &struct {
					ID    string
					Name  string
					Input strings.Builder
				}{
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}
			case "compaction":
				compactionBlocks[event.Index] = &strings.Builder{}
			}

		case "content_block_delta":
			if event.Delta == nil {
				continue
			}
			if acc, ok := compactionBlocks[event.Index]; ok {
				acc.WriteString(event.Delta.Text)
				continue
			}
			if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				if firstChunk {
					spinner.Stop()
					firstChunk = false
					api.PrintAIHeader()
				}
				fmt.Print(event.Delta.Text)
				fullResponse.WriteString(event.Delta.Text)
			}
			if event.Delta.Type == "input_json_delta" {
				if acc := toolUses[event.Index]; acc != nil {
					acc.Input.WriteString(event.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			if acc, ok := compactionBlocks[event.Index]; ok {
				compactionOutput.WriteString(acc.String())
				delete(compactionBlocks, event.Index)
			}
			if acc := toolUses[event.Index]; acc != nil {
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(acc.Input.String()), &input); err == nil {
					if toolJSON, err := claude.ConvertToolUseToToolJSON(acc.ID, acc.Name, input); err == nil {
						toolCallsOutput.WriteString(toolJSON)
					}
				}
			}
		}
	}

done:
	spinner.Stop()

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	if p.usageCallback != nil && lastUsage != nil {
		p.usageCallback(*lastUsage)
	}

	content := fullResponse.String()
	if compactionOutput.Len() > 0 {
		content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
	}

	if toolCallsOutput.Len() > 0 {
		if content != "" {
			fmt.Println()
			return content + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}

	fmt.Println()
	return content, nil
}

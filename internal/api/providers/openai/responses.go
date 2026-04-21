package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultOpenAIResponsesURL = "https://api.openai.com/v1/responses"

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// convertHistoryToResponsesInput は api.ConvertHistoryToInputItems のラッパー
// Responses API 用の InputItem 形式に変換
func convertHistoryToResponsesInput(history []api.Message) []InputItem {
	return api.ConvertHistoryToInputItems(history)
}

// ReasoningConfig は OpenAI Extended Thinking の設定
type ReasoningConfig struct {
	Effort string `json:"effort"` // none, low, medium, high（omitempty削除で明示的に送信）
}

// ResponsesTool は Responses API 用のツール定義
// Chat Completions API と異なり、"function" キーのネストが不要
type ResponsesTool struct {
	Type        string                 `json:"type"`                  // "function"
	Name        string                 `json:"name"`                  // ツール名
	Description string                 `json:"description,omitempty"` // ツールの説明
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // JSON Schema
	Strict      bool                   `json:"strict,omitempty"`      // Structured Output
}

// ResponsesRequest は Responses API リクエスト
type ResponsesRequest struct {
	Model                string           `json:"model"`
	Input                interface{}      `json:"input,omitempty"`                // string or []InputItem（previous_response_id使用時は省略可）
	PreviousResponseID   string           `json:"previous_response_id,omitempty"` // 前回のレスポンスID（キャッシュ用）
	Instructions         string           `json:"instructions,omitempty"`         // システムプロンプト
	MaxOutputTokens      int              `json:"max_output_tokens,omitempty"`    // 最大出力トークン数
	Stream               bool             `json:"stream,omitempty"`
	Store                bool             `json:"store"`                            // レスポンスを保存（previous_response_id に必要）
	Reasoning            *ReasoningConfig `json:"reasoning,omitempty"`              // Extended Thinking
	Tools                []ResponsesTool  `json:"tools,omitempty"`                  // ツール定義
	ToolChoice           interface{}      `json:"tool_choice,omitempty"`            // "auto", "none", "required", またはオブジェクト
	PromptCacheKey       string           `json:"prompt_cache_key,omitempty"`       // プロンプトキャッシュのルーティングキー
	PromptCacheRetention string           `json:"prompt_cache_retention,omitempty"` // キャッシュ保持期間（"24h"でextended cache）
}

// InputItem は api.InputItem のエイリアス（api packageで定義）
type InputItem = api.InputItem

// InputContentPart は api.InputContentPart のエイリアス（api packageで定義）
type InputContentPart = api.InputContentPart

// ResponseMetadata はレスポンスメタデータ（response.created / response.completed イベント用）
type ResponseMetadata struct {
	ID     string          `json:"id"`               // "resp_xxx..."
	Status string          `json:"status,omitempty"` // "in_progress", "completed"
	Model  string          `json:"model,omitempty"`
	Usage  *ResponsesUsage `json:"usage,omitempty"` // response.completed で取得
}

// ResponsesUsage は Responses API の usage 情報
type ResponsesUsage struct {
	InputTokens         int                     `json:"input_tokens"`
	OutputTokens        int                     `json:"output_tokens"`
	InputTokensDetails  *ResponsesInputDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponsesOutputDetails `json:"output_tokens_details,omitempty"`
}

// ResponsesInputDetails は Responses API の入力トークン詳細
type ResponsesInputDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// ResponsesOutputDetails は Responses API の出力トークン詳細
type ResponsesOutputDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// ResponsesError は Responses API のエラー情報
type ResponsesError struct {
	Type    string `json:"type"`              // "insufficient_quota", "rate_limit_exceeded" 等
	Code    string `json:"code"`              // エラーコード
	Message string `json:"message,omitempty"` // エラーメッセージ
}

// ResponsesStreamChunk は Responses API ストリーミングチャンク
type ResponsesStreamChunk struct {
	Type     string            `json:"type"`               // "response.output_text.delta", "response.created", etc.
	Delta    string            `json:"delta,omitempty"`    // テキスト差分
	Response *ResponseMetadata `json:"response,omitempty"` // response.created で取得
	Item     *ResponsesItem    `json:"item,omitempty"`     // response.output_item.added で取得（function_call用）
	Usage    *ResponsesUsage   `json:"usage,omitempty"`    // response.completed で取得
	Error    *ResponsesError   `json:"error,omitempty"`    // error イベントで取得
}

// ResponsesItem は output_item のデータ（function_call 等）
type ResponsesItem struct {
	Type      string `json:"type,omitempty"`      // "function_call"
	Name      string `json:"name,omitempty"`      // ツール名
	CallID    string `json:"call_id,omitempty"`   // 呼び出しID
	Arguments string `json:"arguments,omitempty"` // 完了時の引数（response.function_call_arguments.done）
}

// ResponsesResult は Responses API のレスポンス結果（ID付き）
type ResponsesResult struct {
	Content    string // テキストコンテンツ
	ResponseID string // レスポンスID（response.created から取得）
}

// chatWithResponses は Responses API でチャット
// previous_response_id を使用してキャッシュを活用
func (p *Provider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	errOut := api.ErrorWriterFromContext(ctx)
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(errOut, "[DEBUG OpenAI] chatWithResponses called, model=%s\n", model)
	}
	cfg := tools.ConfigFromContext(ctx)

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	reqBody := ResponsesRequest{
		Model:                model,
		MaxOutputTokens:      api.GetMaxOutputTokens(ctx, "openai", model),
		Stream:               true,
		Store:                true,
		Tools:                GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools), // Function Calling
		PromptCacheKey:       BuildPromptCacheKey(model, systemPrompt),
		PromptCacheRetention: "24h",
	}

	// tool_choice 強制設定がある場合
	if p.toolChoice != nil {
		reqBody.ToolChoice = map[string]interface{}{
			"type": "function",
			"function": map[string]string{
				"name": *p.toolChoice,
			},
		}
	}

	// システムプロンプトを developer メッセージとして Input の先頭に追加
	// （Instructions フィールドだと Prompt Cache が効かないため）
	developerMsg := InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}

	// previous_response_id がある場合はキャッシュを活用
	if p.lastResponseID != "" && len(history) > 0 {
		lastMsg := history[len(history)-1]

		if lastMsg.Role == "tool" {
			// Function Calling 結果: previous_response_id + 末尾の tool 結果のみ送信
			// （parallel tool calls 対応: 末尾から連続する tool をすべて送る）
			reqBody.PreviousResponseID = p.lastResponseID
			toolStart := len(history) - 1
			for toolStart >= 0 && history[toolStart].Role == "tool" {
				toolStart--
			}
			toolMessages := history[toolStart+1:]
			toolOutputs := make([]InputItem, 0, len(toolMessages))
			for _, msg := range toolMessages {
				toolOutputs = append(toolOutputs, InputItem{
					Type:   "function_call_output",
					CallID: msg.ToolCallID,
					Output: msg.Content,
				})
			}
			reqBody.Input = toolOutputs
		} else {
			// 通常メッセージ: previous_response_id で最新メッセージのみ
			// NOTE: previous_response_id 使用時は developer メッセージ不要（前回と同じなので）
			reqBody.PreviousResponseID = p.lastResponseID
			reqBody.Input = []InputItem{{
				Type:    "message",
				Role:    lastMsg.Role,
				Content: lastMsg.Content,
			}}
		}
	} else {
		// 初回または responseID がない場合は履歴全体を送信
		historyInput := convertHistoryToResponsesInput(history)
		reqBody.Input = append([]InputItem{developerMsg}, historyInput...)
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else if isCodexModel(model) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "low",
		}
	}
	// 非Codexモデルでthinking無効の場合はreasoningフィールドを送信しない（nil）

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(errOut, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// previous_response_id が無効な場合のフォールバック
	if resp.StatusCode == http.StatusBadRequest && p.lastResponseID != "" {
		spinner.Stop()
		// responseID をクリアしてリトライ
		p.lastResponseID = ""
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		p.lastResponseID = responseID
	}
	return content, err
}

// responsesFunctionCallAccumulator は Responses API の function_call を累積
type responsesFunctionCallAccumulator struct {
	CallID    string
	Name      string
	Arguments strings.Builder
}

type responsesStreamState struct {
	spinner       *ui.Spinner
	errOut        io.Writer
	debug         bool
	responseID    string
	functionCalls map[string]*responsesFunctionCallAccumulator
	toolCallsOut  strings.Builder
	lastUsage     *api.Usage
}

func newResponsesStreamState(spinner *ui.Spinner, errOut io.Writer) *responsesStreamState {
	return &responsesStreamState{
		spinner:       spinner,
		errOut:        errOut,
		debug:         os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		functionCalls: make(map[string]*responsesFunctionCallAccumulator),
	}
}

func (s *responsesStreamState) parseLine(line string) (string, bool, error) {
	if s.debug && line != "" {
		s.debugf("[DEBUG OpenAI Responses] SSE line: %s\n", line)
	}

	data, done, handled := parseResponsesSSEDataLine(line)
	if !handled {
		return "", false, nil
	}
	if done {
		return "", true, nil
	}

	chunk, err := decodeResponsesStreamChunk(data)
	if err != nil {
		return "", false, nil // パースエラーはスキップ
	}

	return s.handleChunk(chunk, data)
}

func parseResponsesSSEDataLine(line string) (data string, done bool, handled bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, false
	}

	data = strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, true
	}

	return data, false, true
}

func decodeResponsesStreamChunk(data string) (ResponsesStreamChunk, error) {
	var chunk ResponsesStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ResponsesStreamChunk{}, err
	}
	return chunk, nil
}

func (s *responsesStreamState) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}
	fmt.Fprintf(s.errOut, format, args...)
}

func (s *responsesStreamState) handleChunk(chunk ResponsesStreamChunk, rawData string) (string, bool, error) {
	s.debugf("[DEBUG OpenAI Responses] event: %s\n", chunk.Type)
	if chunk.Type == "response.completed" {
		s.debugf("[DEBUG OpenAI Responses] raw data: %s\n", rawData)
	}

	if chunk.Type == "response.created" && chunk.Response != nil {
		s.responseID = chunk.Response.ID
	}

	if chunk.Type == "error" {
		errMsg := "OpenAI API error"
		if chunk.Error != nil {
			if chunk.Error.Message != "" {
				errMsg = chunk.Error.Message
			} else if chunk.Error.Code != "" {
				errMsg = fmt.Sprintf("OpenAI API error: %s", chunk.Error.Code)
			}
		}
		return "", true, fmt.Errorf("%s", errMsg)
	}

	if chunk.Type == "response.failed" {
		return "", true, fmt.Errorf("OpenAI Responses API request failed")
	}

	if chunk.Type == "response.output_item.added" && chunk.Item != nil && chunk.Item.Type == "function_call" {
		if s.spinner != nil {
			s.spinner.Stop()
			s.spinner.Start(ui.SpinnerMessageForTool(chunk.Item.Name))
		}
		acc := &responsesFunctionCallAccumulator{
			CallID: chunk.Item.CallID,
			Name:   chunk.Item.Name,
		}
		s.functionCalls[chunk.Item.CallID] = acc
	}

	if chunk.Type == "response.function_call_arguments.delta" {
		callID := ""
		if chunk.Item != nil {
			callID = chunk.Item.CallID
		}

		if callID != "" {
			if acc, ok := s.functionCalls[callID]; ok {
				acc.Arguments.WriteString(chunk.Delta)
			}
		} else if len(s.functionCalls) == 1 {
			for _, acc := range s.functionCalls {
				acc.Arguments.WriteString(chunk.Delta)
				break
			}
		}
	}

	if chunk.Type == "response.function_call_arguments.done" && chunk.Item != nil {
		if acc, ok := s.functionCalls[chunk.Item.CallID]; ok {
			if chunk.Item.Arguments != "" {
				acc.Arguments.Reset()
				acc.Arguments.WriteString(chunk.Item.Arguments)
			}
		}
	}

	if chunk.Type == "response.output_text.delta" {
		return chunk.Delta, false, nil
	}

	if chunk.Type == "response.completed" || chunk.Type == "response.done" {
		s.captureUsage(chunk)
		s.appendFunctionCallsToOutput()
		return "", true, nil
	}

	return "", false, nil
}

func (s *responsesStreamState) captureUsage(chunk ResponsesStreamChunk) {
	var usage *ResponsesUsage
	if chunk.Response != nil && chunk.Response.Usage != nil {
		usage = chunk.Response.Usage
	} else if chunk.Usage != nil {
		usage = chunk.Usage
	}

	if usage == nil {
		s.debugf("[DEBUG OpenAI Responses] %s event but usage is nil\n", chunk.Type)
		return
	}

	cachedTokens := 0
	if usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if usage.OutputTokensDetails != nil {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	s.lastUsage = &api.Usage{
		InputTokens:       usage.InputTokens,
		OutputTokens:      usage.OutputTokens,
		ThinkingTokens:    reasoningTokens,
		CachedInputTokens: cachedTokens,
	}
	s.debugf("[DEBUG OpenAI Responses] usage received: input=%d, output=%d, cached=%d\n",
		usage.InputTokens, usage.OutputTokens, cachedTokens)
}

func (s *responsesStreamState) appendFunctionCallsToOutput() {
	for _, acc := range s.functionCalls {
		tc := &api.OpenAIToolCall{
			ID:   acc.CallID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      acc.Name,
				Arguments: acc.Arguments.String(),
			},
		}
		if toolJSON, err := ConvertToolCallToToolJSON(tc); err == nil {
			s.toolCallsOut.WriteString(toolJSON)
		}
	}
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却（content, responseID, error）
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	state := newResponsesStreamState(spinner, api.ErrorWriterFromContext(ctx))
	content, err := api.ParseStreamingResponse(ctx, resp, spinner, state.parseLine)
	if err != nil {
		return "", state.responseID, err
	}

	// usage コールバックを呼び出し
	if state.lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*state.lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if state.toolCallsOut.Len() > 0 {
		if content != "" {
			return content + state.toolCallsOut.String(), state.responseID, nil
		}
		return state.toolCallsOut.String(), state.responseID, nil
	}
	return content, state.responseID, nil
}

// chatWithImageResponses は Responses API で画像付きメッセージを処理
// NOTE: 画像付きの場合は previous_response_id を使用しない（キャッシュ動作が不明瞭なため）
func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	cfg := tools.ConfigFromContext(ctx)

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	// システムプロンプトを developer メッセージとして Input の先頭に追加
	// （Instructions フィールドだと Prompt Cache が効かないため）
	developerMsg := InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}

	// 入力を構築（developer + 履歴）
	input := append([]InputItem{developerMsg}, convertHistoryToResponsesInput(history)...)

	// 画像付きユーザーメッセージを追加
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)
	imageMessage := InputItem{
		Type: "message",
		Role: "user",
		Content: []InputContentPart{
			{
				Type:     "input_image",
				ImageURL: dataURL,
			},
			{
				Type: "input_text",
				Text: userMessage,
			},
		},
	}
	input = append(input, imageMessage)

	reqBody := ResponsesRequest{
		Model:                model,
		Input:                input,
		MaxOutputTokens:      api.GetMaxOutputTokens(ctx, "openai", model),
		Stream:               true,
		Store:                true,
		Tools:                GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools), // Function Calling
		PromptCacheKey:       BuildPromptCacheKey(model, systemPrompt),
		PromptCacheRetention: "24h",
	}

	// tool_choice 強制設定がある場合
	if p.toolChoice != nil {
		reqBody.ToolChoice = map[string]interface{}{
			"type": "function",
			"function": map[string]string{
				"name": *p.toolChoice,
			},
		}
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else if isCodexModel(model) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "low",
		}
	}
	// 非Codexモデルでthinking無効の場合はreasoningフィールドを送信しない（nil）

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		errOut := api.ErrorWriterFromContext(ctx)
		fmt.Fprintf(errOut, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		// 画像メッセージ後も responseID を保存（次回テキストのみの場合に使用可能）
		p.lastResponseID = responseID
	}
	return content, err
}

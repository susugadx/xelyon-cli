// Package openaicompat は OpenAI 互換 API の request 構築補助を提供する。
package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ChatCompletionsRequest は OpenAI 互換 Chat Completions payload を表す。
type ChatCompletionsRequest struct {
	Model                string             `json:"model"`
	Messages             []api.Message      `json:"messages"`
	MaxTokens            int                `json:"max_tokens,omitempty"`
	MaxCompletionTokens  int                `json:"max_completion_tokens,omitempty"`
	Stream               bool               `json:"stream"`
	StreamOptions        *api.StreamOptions `json:"stream_options,omitempty"`
	ReasoningEffort      string             `json:"reasoning_effort,omitempty"`
	Tools                []api.OpenAITool   `json:"tools,omitempty"`
	ToolChoice           any                `json:"tool_choice,omitempty"`
	PromptCacheKey       string             `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string             `json:"prompt_cache_retention,omitempty"`
	ExtraFields          map[string]any     `json:"-"`
	ImagePayloadMode     ImagePayloadMode   `json:"-"`
}

// FunctionCallingOptions は OpenAI 互換 Function Calling 設定を表す。
type FunctionCallingOptions struct {
	Tools            []api.OpenAITool
	ToolName         *string
	ToolChoicePolicy ToolChoicePolicy
}

// ChatCompletionsRequestOptions は ChatCompletionsRequest の構築入力を表す。
type ChatCompletionsRequestOptions struct {
	Model                string
	Messages             []api.Message
	SystemPrompt         string
	ActiveContext        []api.ActiveContextBlock
	History              []api.Message
	MaxTokens            int
	MaxCompletionTokens  int
	Stream               bool
	StreamOptions        *api.StreamOptions
	IncludeUsage         bool
	ReasoningEffort      string
	InitialToolChoice    any
	FunctionCalling      *FunctionCallingOptions
	PromptCacheKey       string
	PromptCacheRetention string
	ExtraFields          map[string]any
	ImagePayloadMode     ImagePayloadMode
}

// ToolChoicePolicy は provider ごとの tool_choice 方針を表す。
type ToolChoicePolicy func(toolName *string) any

// ImagePayloadMode は OpenAI 互換 payload に履歴画像を含めるかを表す。
type ImagePayloadMode int

const (
	// ImagePayloadTextOnly は画像 state を JSON に出さず、message content だけを送る。
	ImagePayloadTextOnly ImagePayloadMode = iota
	// ImagePayloadMultimodal は画像対応 provider 向けに image_url content part を送る。
	ImagePayloadMultimodal
)

var chatCompletionsStandardFields = map[string]struct{}{
	"model":                  {},
	"messages":               {},
	"max_tokens":             {},
	"max_completion_tokens":  {},
	"stream":                 {},
	"stream_options":         {},
	"reasoning_effort":       {},
	"tools":                  {},
	"tool_choice":            {},
	"prompt_cache_key":       {},
	"prompt_cache_retention": {},
}

// BuildChatCompletionsRequest は OpenAI 互換 Chat Completions payload を構築する。
func BuildChatCompletionsRequest(options ChatCompletionsRequestOptions) ChatCompletionsRequest {
	messages := options.Messages
	if messages == nil {
		messages = BuildChatMessagesWithActiveContext(options.SystemPrompt, options.ActiveContext, options.History)
	}

	streamOptions := options.StreamOptions
	if streamOptions == nil && options.IncludeUsage {
		streamOptions = &api.StreamOptions{IncludeUsage: true}
	}

	req := ChatCompletionsRequest{
		Model:                options.Model,
		Messages:             messages,
		Stream:               options.Stream,
		StreamOptions:        streamOptions,
		ReasoningEffort:      options.ReasoningEffort,
		ToolChoice:           options.InitialToolChoice,
		PromptCacheKey:       options.PromptCacheKey,
		PromptCacheRetention: options.PromptCacheRetention,
		ExtraFields:          cloneExtraFields(options.ExtraFields),
		ImagePayloadMode:     options.ImagePayloadMode,
	}
	if options.MaxCompletionTokens > 0 {
		req.MaxCompletionTokens = options.MaxCompletionTokens
	} else {
		req.MaxTokens = options.MaxTokens
	}

	if options.FunctionCalling != nil {
		req.ApplyFunctionCalling(*options.FunctionCalling)
	}
	return req
}

// ApplyFunctionCalling は payload に OpenAI 互換の tools と tool_choice を設定する。
func (r *ChatCompletionsRequest) ApplyFunctionCalling(options FunctionCallingOptions) {
	if r == nil {
		return
	}
	r.Tools = options.Tools

	policy := options.ToolChoicePolicy
	if policy == nil {
		policy = DefaultToolChoicePolicy
	}
	r.ToolChoice = policy(options.ToolName)
}

// MarshalJSON は標準 payload に provider 固有 extra fields を衝突検出つきで混ぜる。
func (r ChatCompletionsRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{
		"model":    r.Model,
		"messages": chatCompletionsMessagePayloads(r.Messages, r.ImagePayloadMode),
		"stream":   r.Stream,
	}
	if r.MaxTokens > 0 {
		body["max_tokens"] = r.MaxTokens
	}
	if r.MaxCompletionTokens > 0 {
		body["max_completion_tokens"] = r.MaxCompletionTokens
	}
	if r.StreamOptions != nil {
		body["stream_options"] = r.StreamOptions
	}
	if r.ReasoningEffort != "" {
		body["reasoning_effort"] = r.ReasoningEffort
	}
	if len(r.Tools) > 0 {
		body["tools"] = r.Tools
	}
	if r.ToolChoice != nil {
		body["tool_choice"] = r.ToolChoice
	}
	if r.PromptCacheKey != "" {
		body["prompt_cache_key"] = r.PromptCacheKey
	}
	if r.PromptCacheRetention != "" {
		body["prompt_cache_retention"] = r.PromptCacheRetention
	}

	if len(r.ExtraFields) == 0 {
		return json.Marshal(body)
	}
	for key, value := range r.ExtraFields {
		if _, ok := chatCompletionsStandardFields[key]; ok {
			return nil, fmt.Errorf("extra field %q conflicts with standard chat completions field", key)
		}
		body[key] = value
	}
	return json.Marshal(body)
}

func chatCompletionsMessagePayloads(messages []api.Message, imageMode ImagePayloadMode) []any {
	if messages == nil {
		return nil
	}
	payloads := make([]any, 0, len(messages))
	for _, message := range messages {
		if imageMode == ImagePayloadMultimodal && message.HasImage() {
			payloads = append(payloads, chatCompletionsImageMessagePayload(message))
			continue
		}
		payloads = append(payloads, message)
	}
	return payloads
}

type chatCompletionsImageMessage struct {
	Role             string               `json:"role"`
	Content          []chatContentPart    `json:"content"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
	ToolCalls        []api.OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolName         string               `json:"tool_name,omitempty"`
}

type chatContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *chatImageURL `json:"image_url,omitempty"`
}

type chatImageURL struct {
	URL string `json:"url"`
}

func chatCompletionsImageMessagePayload(message api.Message) chatCompletionsImageMessage {
	image := message.ImageData()
	parts := make([]chatContentPart, 0, 2)
	if message.Content != "" {
		parts = append(parts, chatContentPart{
			Type: "text",
			Text: message.Content,
		})
	}
	if image != nil {
		parts = append(parts, chatContentPart{
			Type: "image_url",
			ImageURL: &chatImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64),
			},
		})
	}
	return chatCompletionsImageMessage{
		Role:             message.Role,
		Content:          parts,
		ReasoningContent: message.ReasoningContent,
		ToolCallID:       message.ToolCallID,
		ToolCalls:        message.ToolCalls,
		ToolName:         message.ToolName,
	}
}

// BuildChatMessages は system prompt と履歴から OpenAI 互換 messages を構築する。
func BuildChatMessages(systemPrompt string, history []api.Message) []api.Message {
	return BuildChatMessagesWithActiveContext(systemPrompt, nil, history)
}

// BuildChatMessagesWithActiveContext は system prompt、active context、履歴から messages を構築する。
func BuildChatMessagesWithActiveContext(systemPrompt string, activeContext []api.ActiveContextBlock, history []api.Message) []api.Message {
	messages := make([]api.Message, 0, len(history)+1)
	messages = append(messages, api.Message{Role: "system", Content: systemPrompt})
	if content := api.RenderActiveContextBlocks(activeContext); content != "" {
		messages = append(messages, api.Message{Role: "system", Content: content})
	}
	messages = append(messages, history...)
	return messages
}

// BuildChatMessageInterfacesWithActiveContext は multimodal message を後続で追加する OpenAI 互換 payload 用の messages を構築する。
// transform は履歴メッセージだけに適用する。
func BuildChatMessageInterfacesWithActiveContext(systemPrompt string, activeContext []api.ActiveContextBlock, history []api.Message, transform func(api.Message) api.Message) []any {
	return BuildChatMessageInterfacesWithActiveContextAndImagePayloadMode(systemPrompt, activeContext, history, transform, ImagePayloadTextOnly)
}

// BuildChatMessageInterfacesWithActiveContextAndImagePayloadMode は履歴画像の projection 方針を指定して messages を構築する。
func BuildChatMessageInterfacesWithActiveContextAndImagePayloadMode(systemPrompt string, activeContext []api.ActiveContextBlock, history []api.Message, transform func(api.Message) api.Message, imageMode ImagePayloadMode) []any {
	prefix := BuildChatMessagesWithActiveContext(systemPrompt, activeContext, nil)
	result := make([]any, 0, len(prefix)+len(history))
	for _, message := range prefix {
		result = append(result, message)
	}
	for _, message := range history {
		if transform != nil {
			message = transform(message)
		}
		if imageMode == ImagePayloadMultimodal && message.HasImage() {
			result = append(result, chatCompletionsImageMessagePayload(message))
			continue
		}
		result = append(result, message)
	}
	return result
}

// DefaultToolChoicePolicy は未指定なら auto、指定ありなら function 強制にする標準方針を返す。
func DefaultToolChoicePolicy(toolName *string) any {
	return AllowForcedToolChoicePolicy(toolName)
}

// AllowForcedToolChoicePolicy は toolName 指定時に function tool_choice を強制する標準方針。
func AllowForcedToolChoicePolicy(toolName *string) any {
	return BuildFunctionToolChoice(toolName)
}

// AutoToolChoicePolicy は provider 制約により forced tool_choice を送れない場合に auto を返す。
func AutoToolChoicePolicy(*string) any {
	return "auto"
}

// BuildFunctionToolChoice は OpenAI 互換 tool_choice を構築する。
func BuildFunctionToolChoice(toolName *string) any {
	if toolName == nil {
		return "auto"
	}
	return map[string]interface{}{
		"type": "function",
		"function": map[string]string{
			"name": *toolName,
		},
	}
}

func cloneExtraFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}

// NewBearerJSONRequest は Bearer 認証つき JSON POST request を作成する。
func NewBearerJSONRequest(ctx context.Context, url, apiKey string, body any) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return NewBearerJSONBytesRequest(ctx, url, apiKey, jsonBody)
}

// NewBearerJSONBytesRequest は marshal 済み JSON payload から Bearer 認証つき POST request を作成する。
func NewBearerJSONBytesRequest(ctx context.Context, url, apiKey string, payload []byte) (*http.Request, error) {
	req, err := NewJSONBytesRequest(ctx, url, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return req, nil
}

// NewAPIKeyJSONRequest は api-key ヘッダー認証つき JSON POST request を作成する。
func NewAPIKeyJSONRequest(ctx context.Context, url, apiKey string, body any) (*http.Request, error) {
	return NewHeaderJSONRequest(ctx, url, "api-key", apiKey, body)
}

// NewHeaderJSONRequest は任意ヘッダー認証つき JSON POST request を作成する。
func NewHeaderJSONRequest(ctx context.Context, url, headerName, headerValue string, body any) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return NewHeaderJSONBytesRequest(ctx, url, headerName, headerValue, jsonBody)
}

// NewHeaderJSONBytesRequest は marshal 済み JSON payload から任意ヘッダー認証つき POST request を作成する。
func NewHeaderJSONBytesRequest(ctx context.Context, url, headerName, headerValue string, payload []byte) (*http.Request, error) {
	req, err := NewJSONBytesRequest(ctx, url, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerName, headerValue)
	return req, nil
}

// NewJSONBytesRequest は marshal 済み JSON payload から認証なし POST request を作成する。
func NewJSONBytesRequest(ctx context.Context, url string, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

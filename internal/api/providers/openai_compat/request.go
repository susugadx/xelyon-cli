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
	Stream               bool               `json:"stream"`
	StreamOptions        *api.StreamOptions `json:"stream_options,omitempty"`
	ReasoningEffort      string             `json:"reasoning_effort,omitempty"`
	Tools                []api.OpenAITool   `json:"tools,omitempty"`
	ToolChoice           any                `json:"tool_choice,omitempty"`
	PromptCacheKey       string             `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string             `json:"prompt_cache_retention,omitempty"`
	ExtraFields          map[string]any     `json:"-"`
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
	History              []api.Message
	MaxTokens            int
	Stream               bool
	StreamOptions        *api.StreamOptions
	IncludeUsage         bool
	ReasoningEffort      string
	InitialToolChoice    any
	FunctionCalling      *FunctionCallingOptions
	PromptCacheKey       string
	PromptCacheRetention string
	ExtraFields          map[string]any
}

// ToolChoicePolicy は provider ごとの tool_choice 方針を表す。
type ToolChoicePolicy func(toolName *string) any

var chatCompletionsStandardFields = map[string]struct{}{
	"model":                  {},
	"messages":               {},
	"max_tokens":             {},
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
		messages = BuildChatMessages(options.SystemPrompt, options.History)
	}

	streamOptions := options.StreamOptions
	if streamOptions == nil && options.IncludeUsage {
		streamOptions = &api.StreamOptions{IncludeUsage: true}
	}

	req := ChatCompletionsRequest{
		Model:                options.Model,
		Messages:             messages,
		MaxTokens:            options.MaxTokens,
		Stream:               options.Stream,
		StreamOptions:        streamOptions,
		ReasoningEffort:      options.ReasoningEffort,
		ToolChoice:           options.InitialToolChoice,
		PromptCacheKey:       options.PromptCacheKey,
		PromptCacheRetention: options.PromptCacheRetention,
		ExtraFields:          cloneExtraFields(options.ExtraFields),
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
	type requestAlias ChatCompletionsRequest
	base, err := json.Marshal(requestAlias(r))
	if err != nil {
		return nil, err
	}

	if len(r.ExtraFields) == 0 {
		return base, nil
	}

	var body map[string]any
	if err := json.Unmarshal(base, &body); err != nil {
		return nil, err
	}
	for key, value := range r.ExtraFields {
		if _, ok := chatCompletionsStandardFields[key]; ok {
			return nil, fmt.Errorf("extra field %q conflicts with standard chat completions field", key)
		}
		body[key] = value
	}
	return json.Marshal(body)
}

// BuildChatMessages は system prompt と履歴から OpenAI 互換 messages を構築する。
func BuildChatMessages(systemPrompt string, history []api.Message) []api.Message {
	messages := make([]api.Message, 0, len(history)+1)
	messages = append(messages, api.Message{Role: "system", Content: systemPrompt})
	messages = append(messages, history...)
	return messages
}

// DefaultToolChoicePolicy は未指定なら auto、指定ありなら function 強制にする標準方針を返す。
func DefaultToolChoicePolicy(toolName *string) any {
	return BuildFunctionToolChoice(toolName)
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

package kimi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

type kimiChatCompletionsBuild struct {
	Model          string
	Request        openaicompat.ChatCompletionsRequest
	ThinkingActive bool
	SpinnerSuffix  string
	PromptCacheKey string
}

type kimiImageChatCompletionsBuild struct {
	Model          string
	Request        kimiMultimodalChatCompletionsRequest
	ThinkingActive bool
	SpinnerSuffix  string
	PromptCacheKey string
}

// kimiChatCompletionsRequestOptions は Kimi 固有の request field を messages から分離して保持する。
type kimiChatCompletionsRequestOptions struct {
	model               string
	maxCompletionTokens int
	stream              bool
	includeUsage        bool
	promptCacheKey      string
	extraFields         map[string]any
	functionCalling     *openaicompat.FunctionCallingOptions
}

// kimiMultimodalChatCompletionsRequest は Kimi Chat Completions の画像入力 payload を表す。
type kimiMultimodalChatCompletionsRequest struct {
	Model               string             `json:"model"`
	Messages            []any              `json:"messages"`
	MaxCompletionTokens int                `json:"max_completion_tokens,omitempty"`
	Stream              bool               `json:"stream"`
	StreamOptions       *api.StreamOptions `json:"stream_options,omitempty"`
	Tools               []api.OpenAITool   `json:"tools,omitempty"`
	ToolChoice          any                `json:"tool_choice,omitempty"`
	PromptCacheKey      string             `json:"prompt_cache_key,omitempty"`
	ExtraFields         map[string]any     `json:"-"`
}

type kimiMultimodalMessage struct {
	Role    string                  `json:"role"`
	Content []kimiMultimodalContent `json:"content"`
}

type kimiMultimodalContent struct {
	Type     string             `json:"type"`
	Text     string             `json:"text,omitempty"`
	ImageURL *kimiImageURLField `json:"image_url,omitempty"`
}

type kimiImageURLField struct {
	URL string `json:"url"`
}

var kimiMultimodalChatCompletionsStandardFields = map[string]struct{}{
	"model":                 {},
	"messages":              {},
	"max_completion_tokens": {},
	"stream":                {},
	"stream_options":        {},
	"tools":                 {},
	"tool_choice":           {},
	"prompt_cache_key":      {},
}

func (p *Provider) buildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) kimiChatCompletionsBuild {
	options, thinkingActive, spinnerSuffix := p.buildChatCompletionsRequestOptions(ctx, systemPrompt, model)
	messages := openaicompat.BuildChatMessages(systemPrompt, history)

	return kimiChatCompletionsBuild{
		Model:          options.model,
		Request:        buildKimiTextChatCompletionsRequest(options, messages),
		ThinkingActive: thinkingActive,
		SpinnerSuffix:  spinnerSuffix,
		PromptCacheKey: options.promptCacheKey,
	}
}

func (p *Provider) buildChatCompletionsRequestOptions(ctx context.Context, systemPrompt string, model string) (kimiChatCompletionsRequestOptions, bool, string) {
	providerConfigKey := p.configLookupKey()
	requestedModel := api.GetDefaultModelWithContext(ctx, model, providerConfigKey, defaultKimiModel)
	extraFields, thinkingActive, spinnerSuffix := kimiThinkingConfig(ctx, providerConfigKey, requestedModel)
	promptCacheKey := buildKimiPromptCacheKey(ctx, requestedModel, systemPrompt)

	return kimiChatCompletionsRequestOptions{
		model:               requestedModel,
		maxCompletionTokens: api.GetMaxOutputTokens(ctx, providerConfigKey, requestedModel),
		stream:              true,
		includeUsage:        true,
		promptCacheKey:      promptCacheKey,
		extraFields:         extraFields,
		functionCalling:     p.buildFunctionCallingOptions(ctx, thinkingActive),
	}, thinkingActive, spinnerSuffix
}

func buildKimiTextChatCompletionsRequest(options kimiChatCompletionsRequestOptions, messages []api.Message) openaicompat.ChatCompletionsRequest {
	return openaicompat.BuildChatCompletionsRequest(options.openAICompatOptions(messages))
}

func (p *Provider) buildImageChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (kimiImageChatCompletionsBuild, error) {
	options, thinkingActive, spinnerSuffix := p.buildChatCompletionsRequestOptions(ctx, systemPrompt, model)
	request, err := buildKimiImageChatCompletionsRequest(options, systemPrompt, history, userMessage, image)
	if err != nil {
		return kimiImageChatCompletionsBuild{}, err
	}

	return kimiImageChatCompletionsBuild{
		Model:          options.model,
		Request:        request,
		ThinkingActive: thinkingActive,
		SpinnerSuffix:  spinnerSuffix,
		PromptCacheKey: options.promptCacheKey,
	}, nil
}

func buildKimiImageChatCompletionsRequest(options kimiChatCompletionsRequestOptions, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData) (kimiMultimodalChatCompletionsRequest, error) {
	dataURL, err := kimiImageDataURL(image)
	if err != nil {
		return kimiMultimodalChatCompletionsRequest{}, err
	}

	messages := make([]any, 0, len(history)+2)
	messages = append(messages, api.Message{Role: "system", Content: systemPrompt})
	for _, msg := range history {
		messages = append(messages, msg)
	}
	messages = append(messages, kimiMultimodalMessage{
		Role: "user",
		Content: []kimiMultimodalContent{
			{
				Type: "text",
				Text: userMessage,
			},
			{
				Type: "image_url",
				ImageURL: &kimiImageURLField{
					URL: dataURL,
				},
			},
		},
	})

	return newKimiMultimodalChatCompletionsRequest(options, messages), nil
}

func (o kimiChatCompletionsRequestOptions) openAICompatOptions(messages []api.Message) openaicompat.ChatCompletionsRequestOptions {
	return openaicompat.ChatCompletionsRequestOptions{
		Model:               o.model,
		Messages:            messages,
		MaxCompletionTokens: o.maxCompletionTokens,
		Stream:              o.stream,
		IncludeUsage:        o.includeUsage,
		PromptCacheKey:      o.promptCacheKey,
		ExtraFields:         o.extraFields,
		FunctionCalling:     o.functionCalling,
	}
}

func newKimiMultimodalChatCompletionsRequest(options kimiChatCompletionsRequestOptions, messages []any) kimiMultimodalChatCompletionsRequest {
	req := kimiMultimodalChatCompletionsRequest{
		Model:               options.model,
		Messages:            messages,
		MaxCompletionTokens: options.maxCompletionTokens,
		Stream:              options.stream,
		PromptCacheKey:      options.promptCacheKey,
		ExtraFields:         cloneKimiExtraFields(options.extraFields),
	}
	if options.includeUsage {
		req.StreamOptions = &api.StreamOptions{IncludeUsage: true}
	}
	if options.functionCalling != nil {
		req.applyFunctionCalling(*options.functionCalling)
	}
	return req
}

func (r *kimiMultimodalChatCompletionsRequest) applyFunctionCalling(options openaicompat.FunctionCallingOptions) {
	if r == nil {
		return
	}
	r.Tools = options.Tools

	policy := options.ToolChoicePolicy
	if policy == nil {
		policy = openaicompat.DefaultToolChoicePolicy
	}
	r.ToolChoice = policy(options.ToolName)
}

// MarshalJSON は Kimi 固有 extra fields を multimodal payload に混ぜる。
func (r kimiMultimodalChatCompletionsRequest) MarshalJSON() ([]byte, error) {
	type requestAlias kimiMultimodalChatCompletionsRequest
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
		if _, ok := kimiMultimodalChatCompletionsStandardFields[key]; ok {
			return nil, fmt.Errorf("extra field %q conflicts with Kimi chat completions field", key)
		}
		body[key] = value
	}
	return json.Marshal(body)
}

func (p *Provider) buildFunctionCallingOptions(ctx context.Context, thinkingActive bool) *openaicompat.FunctionCallingOptions {
	if !p.IsFunctionCallingEnabled() {
		return nil
	}
	return &openaicompat.FunctionCallingOptions{
		Tools:            openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
		ToolName:         p.toolChoice,
		ToolChoicePolicy: kimiToolChoicePolicy(thinkingActive),
	}
}

func kimiToolChoicePolicy(thinkingActive bool) openaicompat.ToolChoicePolicy {
	if thinkingActive {
		return openaicompat.AutoToolChoicePolicy
	}
	return openaicompat.AllowForcedToolChoicePolicy
}

func kimiImageDataURL(image *api.ImageData) (string, error) {
	if image == nil {
		return "", fmt.Errorf("Kimi image input requires non-empty image data")
	}
	encoded := strings.TrimSpace(image.Base64)
	if encoded == "" {
		return "", fmt.Errorf("Kimi image input requires non-empty image data")
	}
	mediaType := strings.ToLower(strings.TrimSpace(image.MediaType))
	if !isKimiSupportedImageMediaType(mediaType) {
		return "", fmt.Errorf("unsupported Kimi image media type %q (supported: image/png, image/jpeg, image/webp, image/gif)", image.MediaType)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid Kimi image base64 data: %w", err)
	}
	if len(decoded) == 0 {
		return "", fmt.Errorf("Kimi image input requires non-empty decoded image data")
	}
	if len(decoded) > api.MaxImageSize || image.Size > api.MaxImageSize {
		size := int64(len(decoded))
		if image.Size > 0 {
			size = image.Size
		}
		return "", fmt.Errorf("Kimi image input is too large: %d bytes (max: %d bytes)", size, api.MaxImageSize)
	}
	if !isKimiImageBytesForMediaType(mediaType, decoded) {
		return "", fmt.Errorf("invalid Kimi image bytes for media type %q", mediaType)
	}
	return fmt.Sprintf("data:%s;base64,%s", mediaType, encoded), nil
}

func isKimiSupportedImageMediaType(mediaType string) bool {
	switch mediaType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func isKimiImageBytesForMediaType(mediaType string, data []byte) bool {
	switch mediaType {
	case "image/png":
		return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
	case "image/jpeg":
		return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
	case "image/gif":
		return bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a"))
	case "image/webp":
		return len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
	default:
		return false
	}
}

func cloneKimiExtraFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}

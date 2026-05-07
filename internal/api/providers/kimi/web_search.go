package kimi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
)

const (
	kimiWebSearchToolName     = "$web_search"
	kimiWebSearchMaxRequests  = 3
	kimiWebSearchSystemPrompt = "You are Kimi. Use the built-in web search tool to answer with concise findings and source URLs when available."
)

type kimiWebSearchRequest struct {
	Model               string                 `json:"model"`
	Messages            []kimiWebSearchMessage `json:"messages"`
	MaxCompletionTokens int                    `json:"max_completion_tokens,omitempty"`
	Stream              bool                   `json:"stream"`
	StreamOptions       *api.StreamOptions     `json:"stream_options,omitempty"`
	Tools               []kimiWebSearchTool    `json:"tools"`
	PromptCacheKey      string                 `json:"prompt_cache_key,omitempty"`
	Thinking            map[string]any         `json:"thinking,omitempty"`
}

type kimiWebSearchMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []api.OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	Name       string               `json:"name,omitempty"`
}

type kimiWebSearchTool struct {
	Type     string                `json:"type"`
	Function kimiWebSearchFunction `json:"function"`
}

type kimiWebSearchFunction struct {
	Name string `json:"name"`
}

type kimiWebSearchStreamResult struct {
	Content      string
	FinishReason string
	ToolCalls    []api.OpenAIToolCall
	Usage        *api.Usage
}

type kimiWebSearchToolCallCollector struct {
	calls map[int]*kimiWebSearchToolCallState
}

type kimiWebSearchToolCallState struct {
	id        string
	callType  string
	name      string
	arguments strings.Builder
}

func registerWebSearch(providerKey string) {
	websearch.RegisterWithContext(providerKey, func(ctx context.Context, query, model string) (string, error) {
		return webSearchWithContextForProvider(ctx, providerKey, query, model)
	})
}

// WebSearchWithContext は request context を使って Kimi built-in $web_search を実行する。
func WebSearchWithContext(ctx context.Context, query, model string) (string, error) {
	return webSearchWithContextForProvider(ctx, "kimi", query, model)
}

func webSearchWithContextForProvider(ctx context.Context, providerConfigKey, query, model string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	apiKey := os.Getenv(kimiAPIKeyEnv)
	if apiKey == "" {
		return "", fmt.Errorf("%s not set", kimiAPIKeyEnv)
	}

	provider := newProvider(apiKey, providerConfigKey)
	provider.SetUsageCallback(websearch.UsageCallbackFromContext(ctx))
	return provider.webSearch(ctx, query, model, providerConfigKey)
}

func (p *Provider) webSearch(ctx context.Context, query, model, providerConfigKey string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("kimi provider is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)

	messages := initialKimiWebSearchMessages(query)
	for requestNumber := 1; requestNumber <= kimiWebSearchMaxRequests; requestNumber++ {
		requestBody := buildKimiWebSearchRequest(ctx, messages, model, providerConfigKey)
		result, err := p.executeWebSearchRequest(ctx, requestBody)
		if err != nil {
			return "", err
		}
		if result.Usage != nil && p.usageCallback != nil {
			p.usageCallback(*result.Usage)
		}

		switch result.FinishReason {
		case "tool_calls":
			if requestNumber == kimiWebSearchMaxRequests {
				return "", fmt.Errorf("kimi %s did not complete within %d requests", kimiWebSearchToolName, kimiWebSearchMaxRequests)
			}
			if len(result.ToolCalls) == 0 {
				return "", fmt.Errorf("kimi %s response finished with tool_calls but returned no tool calls", kimiWebSearchToolName)
			}
			nextMessages, err := appendKimiWebSearchToolLoopMessages(messages, result)
			if err != nil {
				return "", err
			}
			messages = nextMessages
			continue
		case "stop":
			return completeKimiWebSearchContent(result), nil
		default:
			return "", fmt.Errorf("kimi %s ended with unsupported finish_reason %q", kimiWebSearchToolName, result.FinishReason)
		}
	}

	return "", fmt.Errorf("kimi %s did not complete within %d requests", kimiWebSearchToolName, kimiWebSearchMaxRequests)
}

func completeKimiWebSearchContent(result kimiWebSearchStreamResult) string {
	content := strings.TrimSpace(result.Content)
	if content == "" {
		return "No results found."
	}
	return content
}

func buildKimiWebSearchRequest(ctx context.Context, messages []kimiWebSearchMessage, model, providerConfigKey string) kimiWebSearchRequest {
	providerConfigKey = strings.TrimSpace(providerConfigKey)
	if providerConfigKey == "" {
		providerConfigKey = "kimi"
	}
	requestedModel := api.GetDefaultModelWithContext(ctx, model, providerConfigKey, defaultKimiModel)
	thinking := kimiWebSearchThinkingField(ctx, providerConfigKey, requestedModel)

	return kimiWebSearchRequest{
		Model:               requestedModel,
		Messages:            messages,
		MaxCompletionTokens: api.GetMaxOutputTokens(ctx, providerConfigKey, requestedModel),
		Stream:              true,
		StreamOptions:       &api.StreamOptions{IncludeUsage: true},
		Tools:               kimiWebSearchTools(),
		PromptCacheKey:      buildKimiPromptCacheKey(ctx, requestedModel, kimiWebSearchSystemPrompt),
		Thinking:            thinking,
	}
}

func kimiWebSearchThinkingField(ctx context.Context, providerConfigKey, requestedModel string) map[string]any {
	catalogModel := kimiCatalogModel(ctx, providerConfigKey, requestedModel)
	if isKimiForcedThinkingModel(requestedModel) || isKimiForcedThinkingModel(catalogModel) {
		return nil
	}
	return map[string]any{"type": "disabled"}
}

func initialKimiWebSearchMessages(query string) []kimiWebSearchMessage {
	return []kimiWebSearchMessage{
		{Role: "system", Content: kimiWebSearchSystemPrompt},
		{Role: "user", Content: buildKimiWebSearchPrompt(query)},
	}
}

func buildKimiWebSearchPrompt(query string) string {
	return fmt.Sprintf("Use web search to answer the query below. Return a concise summary of the most relevant findings and include source URLs when available.\n\nQuery: %s", query)
}

func kimiWebSearchTools() []kimiWebSearchTool {
	return []kimiWebSearchTool{{
		Type: "builtin_function",
		Function: kimiWebSearchFunction{
			Name: kimiWebSearchToolName,
		},
	}}
}

func appendKimiWebSearchToolLoopMessages(messages []kimiWebSearchMessage, result kimiWebSearchStreamResult) ([]kimiWebSearchMessage, error) {
	next := make([]kimiWebSearchMessage, 0, len(messages)+1+len(result.ToolCalls))
	next = append(next, messages...)
	next = append(next, kimiWebSearchMessage{
		Role:      "assistant",
		Content:   result.Content,
		ToolCalls: result.ToolCalls,
	})

	for _, toolCall := range result.ToolCalls {
		if strings.TrimSpace(toolCall.ID) == "" {
			return nil, fmt.Errorf("kimi %s response returned a tool call without id", kimiWebSearchToolName)
		}
		if toolCall.Function.Name != kimiWebSearchToolName {
			return nil, fmt.Errorf("kimi %s response returned unexpected tool %q", kimiWebSearchToolName, toolCall.Function.Name)
		}
		next = append(next, kimiWebSearchMessage{
			Role:       "tool",
			Content:    toolCall.Function.Arguments,
			ToolCallID: toolCall.ID,
			Name:       toolCall.Function.Name,
		})
	}

	return next, nil
}

func (p *Provider) executeWebSearchRequest(ctx context.Context, body kimiWebSearchRequest) (kimiWebSearchStreamResult, error) {
	req, err := openaicompat.NewBearerJSONRequest(ctx, p.BaseProvider.APIURL, p.APIKey, body)
	if err != nil {
		return kimiWebSearchStreamResult{}, err
	}

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		return kimiWebSearchStreamResult{}, fmt.Errorf("kimi API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return kimiWebSearchStreamResult{}, api.HandleHTTPError(resp, nil, "Kimi")
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return kimiWebSearchStreamResult{}, fmt.Errorf("kimi %s expected streaming response, got %q", kimiWebSearchToolName, resp.Header.Get("Content-Type"))
	}

	return parseKimiWebSearchStream(ctx, resp)
}

func parseKimiWebSearchStream(ctx context.Context, resp *http.Response) (kimiWebSearchStreamResult, error) {
	collector := newKimiWebSearchToolCallCollector()
	var finishReason string
	var lastUsage *api.Usage

	parser := func(line string) (string, bool, error) {
		data, done, handled := openaicompatstream.ParseSSEDataLine(line)
		if !handled {
			return "", false, nil
		}
		if done {
			return "", true, nil
		}
		if err := api.ValidateStreamResponse([]byte(data)); err != nil {
			return "", false, fmt.Errorf("invalid response structure: %w", err)
		}

		chunk, err := openaicompatstream.DecodeChunk(data)
		if err != nil {
			return "", false, err
		}
		usage, err := decodeKimiUsage(chunk.UsagePayload())
		if err != nil {
			return "", false, err
		}
		if usage != nil {
			lastUsage = usage
		}
		if len(chunk.Choices) == 0 {
			return "", false, nil
		}

		choice := chunk.Choices[0]
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
		collector.Append(choice.Delta.ToolCalls)
		return choice.Delta.Content, false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, nil, parser)
	if err != nil {
		return kimiWebSearchStreamResult{}, err
	}
	return kimiWebSearchStreamResult{
		Content:      content,
		FinishReason: finishReason,
		ToolCalls:    collector.ToOpenAIToolCalls(),
		Usage:        lastUsage,
	}, nil
}

func newKimiWebSearchToolCallCollector() *kimiWebSearchToolCallCollector {
	return &kimiWebSearchToolCallCollector{calls: make(map[int]*kimiWebSearchToolCallState)}
}

func (c *kimiWebSearchToolCallCollector) Append(toolCalls []api.OpenAIToolCall) {
	for _, toolCall := range toolCalls {
		acc, ok := c.calls[toolCall.Index]
		if !ok {
			acc = &kimiWebSearchToolCallState{}
			c.calls[toolCall.Index] = acc
		}
		if toolCall.ID != "" {
			acc.id = toolCall.ID
		}
		if toolCall.Type != "" {
			acc.callType = toolCall.Type
		}
		if toolCall.Function.Name != "" {
			acc.name = toolCall.Function.Name
		}
		if toolCall.Function.Arguments != "" {
			acc.arguments.WriteString(toolCall.Function.Arguments)
		}
	}
}

func (c *kimiWebSearchToolCallCollector) ToOpenAIToolCalls() []api.OpenAIToolCall {
	if len(c.calls) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(c.calls))
	for index := range c.calls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	out := make([]api.OpenAIToolCall, 0, len(indexes))
	for _, index := range indexes {
		acc := c.calls[index]
		callType := acc.callType
		if callType == "" {
			callType = "builtin_function"
		}
		out = append(out, api.OpenAIToolCall{
			Index: index,
			ID:    acc.id,
			Type:  callType,
			Function: api.OpenAIToolCallFunction{
				Name:      acc.name,
				Arguments: acc.arguments.String(),
			},
		})
	}
	return out
}

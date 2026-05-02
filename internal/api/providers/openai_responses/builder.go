package openairesponses

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// BaseRequestOptions は Responses API request の共通設定を表す。
type BaseRequestOptions struct {
	Model                                 ModelIdentity
	MaxOutputTokens                       int
	Stream                                bool
	Store                                 bool
	Tools                                 []Tool
	ToolChoice                            interface{}
	Reasoning                             *ReasoningConfig
	PromptCacheKey                        string
	PromptCacheRetention                  string
	ContextManagement                     []ContextManagementSetting
	SkipLocalAutoCompressionAfterResponse bool
}

// ChatRequestOptions は text chat 用 Responses API request の構築入力を表す。
type ChatRequestOptions struct {
	Base               BaseRequestOptions
	SystemPrompt       string
	CompactedInput     []api.InputItem
	History            []api.Message
	PreviousResponseID string
}

// ImageRequestOptions は image input 用 Responses API request の構築入力を表す。
type ImageRequestOptions struct {
	Base           BaseRequestOptions
	SystemPrompt   string
	CompactedInput []api.InputItem
	History        []api.Message
	UserMessage    string
	Image          *api.ImageData
}

// BuildChatRequest は履歴と response id から text chat 用 Responses API request を構築する。
func BuildChatRequest(options ChatRequestOptions) Request {
	reqBody := BuildBaseRequest(options.Base)
	developerMsg := BuildDeveloperMessage(options.SystemPrompt)

	if options.PreviousResponseID != "" && len(options.History) > 0 {
		lastMsg := options.History[len(options.History)-1]
		reqBody.PreviousResponseID = options.PreviousResponseID
		if lastMsg.Role == "tool" {
			reqBody.Input = BuildTrailingToolOutputs(options.History)
			return reqBody
		}

		reqBody.Input = []InputItem{{
			Type:    "message",
			Role:    lastMsg.Role,
			Content: lastMsg.Content,
		}}
		return reqBody
	}

	reqBody.Input = BuildInitialInput(developerMsg, options.CompactedInput, options.History)
	return reqBody
}

// BuildImageRequest は画像入力用 Responses API request を構築する。
func BuildImageRequest(options ImageRequestOptions) Request {
	reqBody := BuildBaseRequest(options.Base)
	input := BuildInitialInput(BuildDeveloperMessage(options.SystemPrompt), options.CompactedInput, options.History)

	if options.Image != nil {
		dataURL := fmt.Sprintf("data:%s;base64,%s", options.Image.MediaType, options.Image.Base64)
		input = append(input, InputItem{
			Type: "message",
			Role: "user",
			Content: []InputContentPart{
				{
					Type:     "input_image",
					ImageURL: dataURL,
				},
				{
					Type: "input_text",
					Text: options.UserMessage,
				},
			},
		})
	}

	reqBody.Input = input
	return reqBody
}

// BuildInitialInput は previous_response_id を使わない request の input を構築する。
func BuildInitialInput(developerMsg InputItem, compactedInput []api.InputItem, history []api.Message) []InputItem {
	input := []InputItem{developerMsg}
	input = append(input, compactedInput...)
	input = append(input, api.ConvertHistoryToInputItems(history)...)
	return input
}

// BuildBaseRequest は provider 差分を渡して Responses API request の共通部を構築する。
func BuildBaseRequest(options BaseRequestOptions) Request {
	return Request{
		Model:                                 options.Model.RequestName(),
		MaxOutputTokens:                       options.MaxOutputTokens,
		Stream:                                options.Stream,
		Store:                                 options.Store,
		Tools:                                 options.Tools,
		ToolChoice:                            options.ToolChoice,
		Reasoning:                             options.Reasoning,
		PromptCacheKey:                        options.PromptCacheKey,
		PromptCacheRetention:                  options.PromptCacheRetention,
		ContextManagement:                     options.ContextManagement,
		SkipLocalAutoCompressionAfterResponse: options.SkipLocalAutoCompressionAfterResponse,
	}
}

// BuildDeveloperMessage は system prompt を Responses API の developer message に変換する。
func BuildDeveloperMessage(systemPrompt string) InputItem {
	return InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}
}

// BuildTrailingToolOutputs は末尾に連続する tool message を function_call_output に変換する。
func BuildTrailingToolOutputs(history []api.Message) []InputItem {
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
	return toolOutputs
}

// BuildFunctionToolChoice は Responses API の function tool_choice を構築する。
func BuildFunctionToolChoice(toolChoice *string) interface{} {
	if toolChoice == nil {
		return nil
	}
	return map[string]interface{}{
		"type": "function",
		"name": *toolChoice,
	}
}

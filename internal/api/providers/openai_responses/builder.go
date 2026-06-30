package openairesponses

import (
	"context"
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
	Instructions                          string
	ContextManagement                     []ContextManagementSetting
	SkipLocalAutoCompressionAfterResponse bool
}

// ChatRequestOptions は text chat 用 Responses API request の構築入力を表す。
type ChatRequestOptions struct {
	Base               BaseRequestOptions
	RequestContext     context.Context
	SystemPrompt       string
	CompactedInput     []api.InputItem
	ActiveContext      []api.ActiveContextBlock
	History            []api.Message
	PreviousResponseID string
}

// ImageRequestOptions は image input 用 Responses API request の構築入力を表す。
type ImageRequestOptions struct {
	Base           BaseRequestOptions
	SystemPrompt   string
	CompactedInput []api.InputItem
	ActiveContext  []api.ActiveContextBlock
	History        []api.Message
	UserMessage    string
	Image          *api.ImageData
}

// InitialInputOptions は previous_response_id を使わない request input の構築入力を表す。
type InitialInputOptions struct {
	DeveloperMessage InputItem
	CompactedInput   []api.InputItem
	ActiveContext    []api.ActiveContextBlock
	History          []api.Message
}

// BuildChatRequest は履歴と response id から text chat 用 Responses API request を構築する。
func BuildChatRequest(options ChatRequestOptions) Request {
	reqBody := BuildBaseRequest(options.Base)
	developerMsg := BuildDeveloperMessage(options.SystemPrompt)
	previousResponseID := PreviousResponseIDForChatRequest(options.RequestContext, options.PreviousResponseID, options.ActiveContext, options.History)

	if previousResponseID != "" && len(options.History) > 0 {
		lastMsg := options.History[len(options.History)-1]
		reqBody.PreviousResponseID = previousResponseID
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

	reqBody.Input = BuildInitialInput(InitialInputOptions{
		DeveloperMessage: developerMsg,
		CompactedInput:   options.CompactedInput,
		ActiveContext:    options.ActiveContext,
		History:          options.History,
	})
	return reqBody
}

// BuildImageRequest は画像入力用 Responses API request を構築する。
func BuildImageRequest(options ImageRequestOptions) Request {
	reqBody := BuildBaseRequest(options.Base)
	input := BuildInitialInput(InitialInputOptions{
		DeveloperMessage: BuildDeveloperMessage(options.SystemPrompt),
		CompactedInput:   options.CompactedInput,
		ActiveContext:    options.ActiveContext,
		History:          options.History,
	})

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
func BuildInitialInput(options InitialInputOptions) []InputItem {
	input := []InputItem{options.DeveloperMessage}
	input = append(input, options.CompactedInput...)
	input = append(input, buildActiveContextMessages(options.ActiveContext)...)
	input = append(input, api.ConvertHistoryToInputItems(options.History)...)
	return input
}

// HasActiveContext は provider request に送る実体のある active context があるかを返す。
func HasActiveContext(blocks []api.ActiveContextBlock) bool {
	return api.RenderActiveContextBlocks(blocks) != ""
}

func buildActiveContextMessages(blocks []api.ActiveContextBlock) []InputItem {
	content := api.RenderActiveContextBlocks(blocks)
	if content == "" {
		return nil
	}
	return []InputItem{
		{
			Type:    "message",
			Role:    "developer",
			Content: content,
		},
	}
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
		Instructions:                          options.Instructions,
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
		toolOutputs = append(toolOutputs, api.NormalizeInputItemOutput(InputItem{
			Type:   "function_call_output",
			CallID: msg.ToolCallID,
			Output: msg.Content,
		}))
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

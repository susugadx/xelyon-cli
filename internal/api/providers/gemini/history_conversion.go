package gemini

import (
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type geminiTextHistoryPartPolicy int

const (
	omitEmptyTextHistoryPart geminiTextHistoryPartPolicy = iota
	includeEmptyTextHistoryPart
)

func geminiFunctionHistoryContents(history []api.Message, textPolicy geminiTextHistoryPartPolicy) []interface{} {
	contents := make([]interface{}, 0, len(history))
	for _, msg := range history {
		contents = append(contents, geminiFunctionHistoryContent(msg, textPolicy))
	}
	return contents
}

func geminiFunctionHistoryContent(msg api.Message, textPolicy geminiTextHistoryPartPolicy) interface{} {
	switch {
	case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
		return geminiAssistantFunctionCallContent(msg)
	case msg.Role == "tool" && msg.ToolCallID != "":
		return geminiToolFunctionResponseContent(msg)
	default:
		return geminiTextHistoryContent(msg, textPolicy)
	}
}

func geminiAssistantFunctionCallContent(msg api.Message) GeminiGenericContent {
	parts := make([]interface{}, 0, len(msg.ToolCalls)+1)
	if msg.Content != "" {
		parts = append(parts, GeminiPart{Text: msg.Content})
	}
	if len(msg.ToolCalls) > 0 && len(msg.ToolCalls[0].ThoughtParts) > 0 {
		for _, tp := range msg.ToolCalls[0].ThoughtParts {
			geminiPart := make(map[string]any)
			if text, ok := tp["text"].(string); ok && text != "" {
				geminiPart["text"] = text
			}
			if thought, ok := tp["thought"].(bool); ok && thought {
				geminiPart["thought"] = true
			}
			if sig, ok := tp["thought_signature"].(string); ok && sig != "" {
				geminiPart["thoughtSignature"] = sig
			}
			if len(geminiPart) > 0 {
				parts = append(parts, geminiPart)
			}
		}
	}
	for _, tc := range msg.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		parts = append(parts, GeminiFunctionCallPart{
			FunctionCall: GeminiFunctionCallData{
				Name: tc.Function.Name,
				Args: args,
			},
			ThoughtSignature: tc.ThoughtSignature,
		})
	}
	return GeminiGenericContent{
		Parts: parts,
		Role:  "model",
	}
}

func geminiToolFunctionResponseContent(msg api.Message) GeminiGenericContent {
	toolName := msg.ToolName
	if toolName == "" {
		toolName = extractToolNameFromContent(msg.Content)
	}
	return GeminiGenericContent{
		Parts: []interface{}{
			GeminiFunctionResponsePart{
				FunctionResponse: GeminiFunctionResponseData{
					Name: toolName,
					Response: map[string]any{
						"result": msg.Content,
					},
				},
			},
		},
		Role: "user",
	}
}

func geminiTextHistoryContent(msg api.Message, textPolicy geminiTextHistoryPartPolicy) GeminiContent {
	role := "user"
	if msg.Role == "assistant" {
		role = "model"
	}
	parts := []GeminiPart{}
	if msg.Content != "" || textPolicy == includeEmptyTextHistoryPart {
		parts = append(parts, GeminiPart{Text: msg.Content})
	}
	return GeminiContent{
		Parts: parts,
		Role:  role,
	}
}

// extractToolNameFromContent はメッセージ内容からツール名を推定
func extractToolNameFromContent(content string) string {
	if strings.HasPrefix(content, "[Tool Result for ") {
		end := strings.Index(content, "]")
		if end > 17 {
			return content[17:end]
		}
	}
	return "unknown_tool"
}

package toolruntime

import (
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ArgsToJSON は tool call の RawArgs を履歴保存用 JSON 文字列に変換する。
func ArgsToJSON(args map[string]any) string {
	if len(args) == 0 {
		return "{}"
	}
	b, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// FormatTextToolResultContent は text-based tool result 履歴用の本文を組み立てる。
func FormatTextToolResultContent(toolName, result string) string {
	return fmt.Sprintf("[Tool Result for %s]\n%s", toolName, result)
}

// BuildToolResultMessage は function calling/text calling の形式差を吸収して履歴 message を作る。
func BuildToolResultMessage(toolCall *tools.ToolCall, functionContent, textContent string) api.Message {
	if toolCall == nil {
		return api.Message{}
	}
	if toolCall.ID != "" {
		return api.Message{
			Role:       "tool",
			Content:    functionContent,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		}
	}
	return api.Message{
		Role:    "user",
		Content: textContent,
	}
}

package api

import (
	"encoding/json"
	"fmt"
)

// ValidateStreamResponse はストリーミングレスポンスの構造を検証
func ValidateStreamResponse(data []byte) error {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// 必須フィールド: choices
	choices, ok := rawMap["choices"]
	if !ok {
		return fmt.Errorf("missing required field: choices")
	}

	choicesArray, ok := choices.([]interface{})
	if !ok || len(choicesArray) == 0 {
		return fmt.Errorf("choices must be a non-empty array")
	}

	// 各choiceの構造検証
	for i, choice := range choicesArray {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			return fmt.Errorf("choices[%d] must be an object", i)
		}

		// deltaまたはmessageが必要
		_, hasDelta := choiceMap["delta"]
		_, hasMessage := choiceMap["message"]
		if !hasDelta && !hasMessage {
			return fmt.Errorf("choices[%d] must have 'delta' or 'message' field", i)
		}
	}

	return nil
}

// ValidateChatResponse は非ストリーミングレスポンスの構造を検証
func ValidateChatResponse(data []byte) error {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// 必須フィールド: choices
	choices, ok := rawMap["choices"]
	if !ok {
		return fmt.Errorf("missing required field: choices")
	}

	choicesArray, ok := choices.([]interface{})
	if !ok || len(choicesArray) == 0 {
		return fmt.Errorf("choices must be a non-empty array")
	}

	// 各choiceの構造検証
	for i, choice := range choicesArray {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			return fmt.Errorf("choices[%d] must be an object", i)
		}

		// message.contentが必要
		message, ok := choiceMap["message"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("choices[%d].message must be an object", i)
		}

		if _, hasContent := message["content"]; !hasContent {
			return fmt.Errorf("choices[%d].message.content is required", i)
		}
	}

	return nil
}

// ValidateToolCall はツール呼び出しレスポンスの構造を検証
func ValidateToolCall(toolCall interface{}) error {
	tcMap, ok := toolCall.(map[string]interface{})
	if !ok {
		return fmt.Errorf("tool_call must be an object")
	}

	// 必須フィールド: function
	function, ok := tcMap["function"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("tool_call.function must be an object")
	}

	// function.name必須
	if _, hasName := function["name"]; !hasName {
		return fmt.Errorf("tool_call.function.name is required")
	}

	// function.arguments必須
	if _, hasArgs := function["arguments"]; !hasArgs {
		return fmt.Errorf("tool_call.function.arguments is required")
	}

	return nil
}

// ValidateErrorResponse はエラーレスポンスの構造を検証
func ValidateErrorResponse(data []byte) (string, error) {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return "", fmt.Errorf("invalid JSON in error response: %w", err)
	}

	// errorフィールドを探す
	errorField, ok := rawMap["error"]
	if !ok {
		// errorフィールドがない場合は全体を返す
		return string(data), nil
	}

	errorMap, ok := errorField.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", errorField), nil
	}

	// messageフィールドがあれば優先
	if msg, ok := errorMap["message"].(string); ok {
		return msg, nil
	}

	// なければJSON全体
	errorBytes, _ := json.Marshal(errorMap)
	return string(errorBytes), nil
}

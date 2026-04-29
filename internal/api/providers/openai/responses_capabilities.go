package openai

import "strings"

func isGPT55ProModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return model == "gpt-5.5-pro" || strings.HasPrefix(model, "gpt-5.5-pro-")
}

func supportsResponsesStreaming(model string) bool {
	return !isGPT55ProModel(model)
}

// ShouldStreamResponses は Responses API で streaming を使えるモデルか返す。
func ShouldStreamResponses(model string) bool {
	return supportsResponsesStreaming(model)
}

func shouldStreamResponses(model string) bool {
	return ShouldStreamResponses(model)
}

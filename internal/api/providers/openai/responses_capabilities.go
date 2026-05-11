package openai

import "github.com/susugadx/xelyon-cli/internal/providerdiag"

func supportsResponsesStreaming(model string) bool {
	return providerdiag.ShouldStreamResponsesCatalogModel(model)
}

// ShouldStreamResponses は Responses API で streaming を使えるモデルか返す。
func ShouldStreamResponses(model string) bool {
	return supportsResponsesStreaming(model)
}

func shouldStreamResponses(model string) bool {
	return ShouldStreamResponses(model)
}

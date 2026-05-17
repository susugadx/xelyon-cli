package gemini

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestExtractToolNameFromContent(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{content: "[Tool Result for read_file] body", want: "read_file"},
		{content: "[Tool Result for tool_with_suffix]", want: "tool_with_suffix"},
		{content: "plain output", want: "unknown_tool"},
	}
	for _, tt := range tests {
		if got := extractToolNameFromContent(tt.content); got != tt.want {
			t.Fatalf("extractToolNameFromContent(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}

func TestGeminiFunctionHistoryContents_EmptyTextPolicy(t *testing.T) {
	history := []api.Message{{Role: "assistant"}}

	requestContents := geminiFunctionHistoryContents(history, includeEmptyTextHistoryPart)
	requestMessage, ok := requestContents[0].(GeminiContent)
	if !ok || len(requestMessage.Parts) != 1 || requestMessage.Parts[0].Text != "" {
		t.Fatalf("request content = %#v, want one empty text part", requestContents[0])
	}

	cacheContents := geminiFunctionHistoryContents(history, omitEmptyTextHistoryPart)
	cacheMessage, ok := cacheContents[0].(GeminiContent)
	if !ok || len(cacheMessage.Parts) != 0 {
		t.Fatalf("cache content = %#v, want empty parts", cacheContents[0])
	}
}

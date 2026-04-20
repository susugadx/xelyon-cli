package openaicompatstream

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestParseSSEDataLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantData    string
		wantDone    bool
		wantHandled bool
	}{
		{
			name:        "normal data line",
			line:        `data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			wantData:    `{"choices":[{"delta":{"content":"Hello"}}]}`,
			wantDone:    false,
			wantHandled: true,
		},
		{
			name:        "done line",
			line:        "data: [DONE]",
			wantDone:    true,
			wantHandled: true,
		},
		{
			name:        "non data line",
			line:        "event: message",
			wantDone:    false,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData, gotDone, gotHandled := ParseSSEDataLine(tt.line)
			if gotData != tt.wantData || gotDone != tt.wantDone || gotHandled != tt.wantHandled {
				t.Fatalf(
					"ParseSSEDataLine() = (%q, %v, %v), want (%q, %v, %v)",
					gotData, gotDone, gotHandled,
					tt.wantData, tt.wantDone, tt.wantHandled,
				)
			}
		})
	}
}

func TestDecodeChunk(t *testing.T) {
	chunk, err := DecodeChunk(`{"choices":[{"delta":{"content":"Hello"}}]}`)
	if err != nil {
		t.Fatalf("DecodeChunk() error = %v", err)
	}
	if len(chunk.Choices) != 1 || chunk.Choices[0].Delta.Content != "Hello" {
		t.Fatalf("DecodeChunk() = %+v, want one content choice", chunk)
	}
}

func TestToolCallCollector_ReconstructsSplitArguments(t *testing.T) {
	collector := NewToolCallCollector()
	var spinnerHints []string

	collector.Append([]api.OpenAIToolCall{
		{Index: 1, ID: "call_2", Function: api.OpenAIToolCallFunction{Name: "list_dir"}},
		{Index: 0, ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file"}},
	}, nil)

	collector.Append([]api.OpenAIToolCall{
		{Index: 0, Function: api.OpenAIToolCallFunction{Arguments: `{"pa`}},
	}, func(name string) {
		spinnerHints = append(spinnerHints, name)
	})
	collector.Append([]api.OpenAIToolCall{
		{Index: 0, Function: api.OpenAIToolCallFunction{Arguments: `th":"a.txt"}`}},
		{Index: 1, Function: api.OpenAIToolCallFunction{Arguments: `{"path":"."}`}},
	}, func(name string) {
		spinnerHints = append(spinnerHints, name)
	})

	got := collector.ToOpenAIToolCalls()
	if len(got) != 2 {
		t.Fatalf("ToOpenAIToolCalls() len = %d, want 2", len(got))
	}
	if got[0].ID != "call_1" || got[0].Function.Name != "read_file" || got[0].Function.Arguments != `{"path":"a.txt"}` {
		t.Fatalf("first tool_call = %+v, want read_file with reconstructed args", got[0])
	}
	if got[1].ID != "call_2" || got[1].Function.Name != "list_dir" || got[1].Function.Arguments != `{"path":"."}` {
		t.Fatalf("second tool_call = %+v, want list_dir with args", got[1])
	}
	if len(spinnerHints) != 3 {
		t.Fatalf("spinner hint calls = %d, want 3", len(spinnerHints))
	}
}

func TestToolCallCollector_ReplaceAt(t *testing.T) {
	collector := NewToolCallCollector()
	var spinnerHints []string

	collector.ReplaceAt(0, api.OpenAIToolCall{
		ID: "call_1",
		Function: api.OpenAIToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path":"a.txt"}`,
		},
	}, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})
	collector.ReplaceAt(0, api.OpenAIToolCall{
		Function: api.OpenAIToolCallFunction{
			Arguments: `{"path":"b.txt"}`,
		},
	}, func(toolName string) {
		spinnerHints = append(spinnerHints, toolName)
	})

	got := collector.ToOpenAIToolCalls()
	if len(got) != 1 {
		t.Fatalf("ToOpenAIToolCalls() len = %d, want 1", len(got))
	}
	if got[0].ID != "call_1" || got[0].Function.Name != "read_file" || got[0].Function.Arguments != `{"path":"b.txt"}` {
		t.Fatalf("tool_call = %+v, want overwritten arguments", got[0])
	}
	if len(spinnerHints) != 2 {
		t.Fatalf("spinner hint calls = %d, want 2", len(spinnerHints))
	}
}

func TestBuildToolCallJSON(t *testing.T) {
	toolCalls := []api.OpenAIToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path":"a.txt"}`,
			},
		},
	}

	got := BuildToolCallJSON(toolCalls, func(tc *api.OpenAIToolCall) (string, error) {
		b, err := json.Marshal(tc)
		if err != nil {
			return "", err
		}
		return string(b), nil
	})

	var decoded api.OpenAIToolCall
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(got) error = %v", err)
	}
	if decoded.Function.Name != "read_file" || !strings.Contains(decoded.Function.Arguments, "a.txt") {
		t.Fatalf("decoded = %+v, want read_file with a.txt args", decoded)
	}
}

func TestHasUsagePayload(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want bool
	}{
		{name: "empty", raw: nil, want: false},
		{name: "null", raw: json.RawMessage("null"), want: false},
		{name: "null with spaces", raw: json.RawMessage("  null  "), want: false},
		{name: "object", raw: json.RawMessage(`{"prompt_tokens":1}`), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasUsagePayload(tt.raw); got != tt.want {
				t.Fatalf("HasUsagePayload(%q) = %v, want %v", string(tt.raw), got, tt.want)
			}
		})
	}
}

func TestDecodeStandardUsage(t *testing.T) {
	usage, err := DecodeStandardUsage(json.RawMessage(`{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}`))
	if err != nil {
		t.Fatalf("DecodeStandardUsage() error = %v", err)
	}
	if usage == nil || usage.InputTokens != 9 || usage.OutputTokens != 4 || usage.CachedInputTokens != 2 {
		t.Fatalf("DecodeStandardUsage() = %+v, want input=9 output=4 cached=2", usage)
	}
}

func TestDecodeStandardUsage_NullReturnsNil(t *testing.T) {
	usage, err := DecodeStandardUsage(json.RawMessage("null"))
	if err != nil {
		t.Fatalf("DecodeStandardUsage(null) error = %v", err)
	}
	if usage != nil {
		t.Fatalf("DecodeStandardUsage(null) = %+v, want nil", usage)
	}
}

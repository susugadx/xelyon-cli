package openaicompatstream

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Chunk は OpenAI 互換 SSE の最小共通レスポンス構造。
// provider 固有仕様は各 provider 側で扱う。
type Chunk struct {
	Choices []Choice        `json:"choices"`
	Usage   json.RawMessage `json:"usage,omitempty"`
}

// Choice は OpenAI 互換 choice の最小共通構造。
type Choice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta は OpenAI 互換 delta の最小共通構造。
type Delta struct {
	Content          string               `json:"content,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []api.OpenAIToolCall `json:"tool_calls,omitempty"`
}

// ParseSSEDataLine は OpenAI 互換 SSE 行から data 部分を抽出する。
// 戻り値: (data, done, handled)
func ParseSSEDataLine(line string) (string, bool, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, false
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, true
	}

	return data, false, true
}

// DecodeChunk は OpenAI 互換 data JSON を共通チャンクにデコードする。
func DecodeChunk(data string) (Chunk, error) {
	var chunk Chunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return Chunk{}, err
	}
	return chunk, nil
}

// ToolCallCollector は分割送信される tool_calls を index ごとに再構築する。
type ToolCallCollector struct {
	calls map[int]*toolCallState
}

type toolCallState struct {
	id        string
	name      string
	arguments strings.Builder
}

// NewToolCallCollector は空の tool_call collector を返す。
func NewToolCallCollector() *ToolCallCollector {
	return &ToolCallCollector{
		calls: make(map[int]*toolCallState),
	}
}

// Append は delta の tool_calls を collector に反映する。
// arguments 断片が届いたときだけ onArguments を呼ぶ。
func (c *ToolCallCollector) Append(toolCalls []api.OpenAIToolCall, onArguments func(toolName string)) {
	for _, tc := range toolCalls {
		acc, ok := c.calls[tc.Index]
		if !ok {
			acc = &toolCallState{}
			c.calls[tc.Index] = acc
		}
		if tc.ID != "" {
			acc.id = tc.ID
		}
		if tc.Function.Name != "" {
			acc.name = tc.Function.Name
		}
		if tc.Function.Arguments != "" {
			if onArguments != nil {
				onArguments(acc.name)
			}
			acc.arguments.WriteString(tc.Function.Arguments)
		}
	}
}

// ReplaceAt は index の tool_call を「最新状態」で上書きする。
// Ollama のように arguments が毎回フル値で届くストリームで使う。
func (c *ToolCallCollector) ReplaceAt(index int, tc api.OpenAIToolCall, onArguments func(toolName string)) {
	acc, ok := c.calls[index]
	if !ok {
		acc = &toolCallState{}
		c.calls[index] = acc
	}

	if tc.ID != "" {
		acc.id = tc.ID
	}
	if tc.Function.Name != "" {
		acc.name = tc.Function.Name
	}
	if tc.Function.Arguments != "" {
		if onArguments != nil {
			onArguments(acc.name)
		}
		acc.arguments.Reset()
		acc.arguments.WriteString(tc.Function.Arguments)
	}
}

// ToOpenAIToolCalls は index 昇順で再構築済み tool_call 一覧を返す。
func (c *ToolCallCollector) ToOpenAIToolCalls() []api.OpenAIToolCall {
	if len(c.calls) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(c.calls))
	for index := range c.calls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	toolCalls := make([]api.OpenAIToolCall, 0, len(indexes))
	for _, index := range indexes {
		acc := c.calls[index]
		toolCalls = append(toolCalls, api.OpenAIToolCall{
			Index: index,
			ID:    acc.id,
			Type:  "function",
			Function: api.OpenAIToolCallFunction{
				Name:      acc.name,
				Arguments: acc.arguments.String(),
			},
		})
	}
	return toolCalls
}

// BuildToolCallJSON は tool_call 一覧を内部 JSON 形式へ変換する。
func BuildToolCallJSON(toolCalls []api.OpenAIToolCall, encode func(tc *api.OpenAIToolCall) (string, error)) string {
	if len(toolCalls) == 0 || encode == nil {
		return ""
	}

	var out strings.Builder
	for i := range toolCalls {
		tc := toolCalls[i]
		if toolJSON, err := encode(&tc); err == nil {
			out.WriteString(toolJSON)
		}
	}
	return out.String()
}

// HasUsagePayload は usage が null 以外の有効 payload を持つかを返す。
func HasUsagePayload(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return false
	}
	return string(trimmed) != "null"
}

// DecodeStandardUsage は OpenAI 互換 usage（prompt/completion/cached）を抽出する。
func DecodeStandardUsage(raw json.RawMessage) (*api.Usage, error) {
	if !HasUsagePayload(raw) {
		return nil, nil
	}

	var usage api.StreamUsageInfo
	if err := json.Unmarshal(raw, &usage); err != nil {
		return nil, err
	}

	cachedTokens := 0
	if usage.PromptTokensDetails != nil {
		cachedTokens = usage.PromptTokensDetails.CachedTokens
	}

	return &api.Usage{
		InputTokens:       usage.PromptTokens,
		OutputTokens:      usage.CompletionTokens,
		CachedInputTokens: cachedTokens,
	}, nil
}

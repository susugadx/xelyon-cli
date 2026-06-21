package openaicompatstream

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// Chunk は OpenAI 互換 SSE の最小共通レスポンス構造。
// provider 固有仕様は各 provider 側で扱う。
type Chunk struct {
	Choices []Choice        `json:"choices"`
	Usage   json.RawMessage `json:"usage,omitempty"`
}

// Choice は OpenAI 互換 choice の最小共通構造。
type Choice struct {
	Delta        Delta           `json:"delta"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Usage        json.RawMessage `json:"usage,omitempty"`
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

// BuildContentWithToolCalls は assistant content の末尾に内部 tool_call JSON を連結する。
func BuildContentWithToolCalls(content string, toolCalls []api.OpenAIToolCall, encode func(tc *api.OpenAIToolCall) (string, error)) string {
	toolCallsOutput := BuildToolCallJSON(toolCalls, encode)
	if toolCallsOutput == "" {
		return content
	}
	return content + toolCallsOutput
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

	apiUsage := usage.ToUsage()
	return &apiUsage, nil
}

// UsagePayload は top-level usage を優先し、なければ choice-level usage を返す。
func (c Chunk) UsagePayload() json.RawMessage {
	if HasUsagePayload(c.Usage) {
		return c.Usage
	}
	for _, choice := range c.Choices {
		if HasUsagePayload(choice.Usage) {
			return choice.Usage
		}
	}
	return nil
}

// ParseSSEOptions は OpenAI 互換 SSE の共通処理オプション。
type ParseSSEOptions struct {
	// OnChunkDecodeError は chunk decode 失敗時の処理を上書きする。
	// nil を返すとエラーを握り潰して継続する。
	OnChunkDecodeError func(error) error
	// OnUsageDecodeError は usage decode 失敗時の処理を上書きする。
	// nil を返すとエラーを握り潰して継続する。
	OnUsageDecodeError func(error) error
	// UsageDecoder は usage payload の decode 方法を上書きする。
	UsageDecoder func(json.RawMessage) (*api.Usage, error)
	// ValidateData は data JSON の構造検証を行う。
	ValidateData func(string) error
	// OnReasoningContent は reasoning_content 受信時に呼ばれる。
	OnReasoningContent func(content string, first bool)
	// OnReasoningBoundary は reasoning_content から通常 content や tool_calls 終了へ移る時に呼ばれる。
	OnReasoningBoundary func()
	// OnToolCallArguments は tool_call arguments 受信時に呼ばれる。
	OnToolCallArguments func(string)
	// ChoiceHandler は choice 処理を上書きする。
	ChoiceHandler func(choice Choice) (content string, done bool, err error)
	// StopOnToolCallsFinish は finish_reason=tool_calls で終了するかどうか。
	StopOnToolCallsFinish bool
}

// ParseSSEResult は OpenAI 互換 SSE の共通処理結果。
type ParseSSEResult struct {
	Content          string
	ReasoningContent string
	ToolCalls        []api.OpenAIToolCall
	Usage            *api.Usage
}

// ParseSSEStream は OpenAI 互換 SSE を共通処理する。
func ParseSSEStream(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner, options ParseSSEOptions) (*ParseSSEResult, error) {
	collector := NewToolCallCollector()
	var lastUsage *api.Usage
	var reasoningContent strings.Builder
	reasoningActive := false

	usageDecoder := options.UsageDecoder
	if usageDecoder == nil {
		usageDecoder = DecodeStandardUsage
	}

	parser := func(line string) (string, bool, error) {
		data, done, handled := ParseSSEDataLine(line)
		if !handled {
			return "", false, nil
		}
		if done {
			return "", true, nil
		}

		if options.ValidateData != nil {
			if err := options.ValidateData(data); err != nil {
				return "", false, err
			}
		}

		chunk, err := DecodeChunk(data)
		if err != nil {
			if options.OnChunkDecodeError != nil {
				return "", false, options.OnChunkDecodeError(err)
			}
			return "", false, err
		}

		usage, err := usageDecoder(chunk.UsagePayload())
		if err != nil {
			if options.OnUsageDecodeError != nil {
				return "", false, options.OnUsageDecodeError(err)
			}
			return "", false, err
		}
		if usage != nil {
			lastUsage = usage
		}

		if len(chunk.Choices) == 0 {
			return "", false, nil
		}

		choice := chunk.Choices[0]

		if choice.Delta.ReasoningContent != "" {
			first := !reasoningActive
			reasoningActive = true
			reasoningContent.WriteString(choice.Delta.ReasoningContent)
			if options.OnReasoningContent != nil {
				options.OnReasoningContent(choice.Delta.ReasoningContent, first)
			}
		}
		if reasoningActive && (choice.Delta.Content != "" || choice.FinishReason == "tool_calls") {
			if options.OnReasoningBoundary != nil {
				options.OnReasoningBoundary()
			}
			reasoningActive = false
		}

		collector.Append(choice.Delta.ToolCalls, options.OnToolCallArguments)

		if options.ChoiceHandler != nil {
			return options.ChoiceHandler(choice)
		}
		if options.StopOnToolCallsFinish && choice.FinishReason == "tool_calls" {
			return "", true, nil
		}
		return choice.Delta.Content, false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return nil, err
	}

	return &ParseSSEResult{
		Content:          content,
		ReasoningContent: reasoningContent.String(),
		ToolCalls:        collector.ToOpenAIToolCalls(),
		Usage:            lastUsage,
	}, nil
}

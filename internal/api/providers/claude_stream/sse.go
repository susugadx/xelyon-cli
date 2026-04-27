package claudestream

import (
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// Delta は Claude 互換ストリームの差分イベント。
type Delta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Data        string `json:"data,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"` // tool_use の input (input_json_delta)
	StopReason  string `json:"stop_reason,omitempty"`  // message_delta 用
}

// ContentBlock は Claude 互換 content_block_start のブロック情報。
type ContentBlock struct {
	Type      string                 `json:"type"`                // "text" or "tool_use"
	ID        string                 `json:"id,omitempty"`        // tool_use 用
	Name      string                 `json:"name,omitempty"`      // tool_use 用
	Text      string                 `json:"text,omitempty"`      // text 用
	Thinking  string                 `json:"thinking,omitempty"`  // thinking 用
	Signature string                 `json:"signature,omitempty"` // thinking 用
	Data      string                 `json:"data,omitempty"`      // redacted_thinking 用
	Input     map[string]interface{} `json:"input,omitempty"`     // tool_use 用（非ストリーミング）
}

// StreamUsage は Claude 互換ストリーム usage。
type StreamUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// StreamEvent は Claude 互換 SSE イベント。
type StreamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *Delta        `json:"delta,omitempty"`
	Usage        *StreamUsage  `json:"usage,omitempty"` // message_delta 用
}

// ParseSSEDataLine は Claude 互換 SSE 行から data 部分を抽出する。
// 戻り値: (data, handled)
func ParseSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	return strings.TrimPrefix(line, "data: "), true
}

// DecodeEvent は data JSON を StreamEvent にデコードする。
func DecodeEvent(data string) (StreamEvent, error) {
	var event StreamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return StreamEvent{}, err
	}
	return event, nil
}

// DecodeMessageStartUsage は message_start の usage を api.Usage へ正規化する。
func DecodeMessageStartUsage(data string) (*api.Usage, error) {
	var payload struct {
		Message struct {
			Usage StreamUsage `json:"usage"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return nil, err
	}

	u := payload.Message.Usage
	return &api.Usage{
		// API の input_tokens は非キャッシュ分のみ。
		InputTokens:         u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens,
		OutputTokens:        u.OutputTokens,
		CachedInputTokens:   u.CacheReadInputTokens,
		CacheCreationTokens: u.CacheCreationInputTokens,
	}, nil
}

// UpdateUsageFromMessageDelta は message_delta usage を既存 usage へ反映する。
// updateInputFallback=true の場合、input/cached/cache-creation もフォールバック更新する。
func UpdateUsageFromMessageDelta(current *api.Usage, usage *StreamUsage, updateInputFallback bool) *api.Usage {
	if usage == nil {
		return current
	}
	if current == nil {
		current = &api.Usage{}
	}

	current.OutputTokens = usage.OutputTokens
	if updateInputFallback {
		if usage.InputTokens > 0 {
			current.InputTokens = usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
		}
		if usage.CacheReadInputTokens > 0 {
			current.CachedInputTokens = usage.CacheReadInputTokens
		}
		if usage.CacheCreationInputTokens > 0 {
			current.CacheCreationTokens = usage.CacheCreationInputTokens
		}
	}
	return current
}

// HandleContentBlockStart は content_block_start を collector に反映する。
func HandleContentBlockStart(event StreamEvent, toolUses *ToolUseCollector, compaction *CompactionCollector) {
	if event.ContentBlock == nil {
		return
	}
	switch event.ContentBlock.Type {
	case "tool_use":
		if toolUses != nil {
			toolUses.Start(event.Index, event.ContentBlock.ID, event.ContentBlock.Name)
		}
	case "compaction":
		if compaction != nil {
			compaction.Start(event.Index)
		}
	}
}

// HandleContentBlockDelta は content_block_delta を collector に反映し、表示すべき text を返す。
func HandleContentBlockDelta(
	event StreamEvent,
	toolUses *ToolUseCollector,
	compaction *CompactionCollector,
	onToolInput func(toolName string),
) string {
	if event.Delta == nil {
		return ""
	}

	switch event.Delta.Type {
	case "text_delta":
		if compaction != nil && compaction.AppendText(event.Index, event.Delta.Text) {
			return ""
		}
		return event.Delta.Text

	case "input_json_delta":
		if toolUses != nil {
			toolUses.AppendInputDelta(event.Index, event.Delta.PartialJSON, onToolInput)
		}
	}
	return ""
}

// HandleContentBlockStop は content_block_stop を collector に反映し、確定 tool JSON を返す。
func HandleContentBlockStop(
	event StreamEvent,
	toolUses *ToolUseCollector,
	compaction *CompactionCollector,
	encode func(id, name string, input map[string]interface{}) (string, error),
) string {
	if compaction != nil {
		compaction.Stop(event.Index)
	}
	if toolUses == nil {
		return ""
	}
	return toolUses.StopAndEncode(event.Index, encode)
}

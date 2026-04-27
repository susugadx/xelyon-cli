package claudestream

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// ToolUseCollector は Anthropic 互換の tool_use ストリーム断片を index ごとに蓄積する。
type ToolUseCollector struct {
	toolUses map[int]*toolUseState
}

type toolUseState struct {
	id    string
	name  string
	input strings.Builder
}

// NewToolUseCollector は空の collector を返す。
func NewToolUseCollector() *ToolUseCollector {
	return &ToolUseCollector{
		toolUses: make(map[int]*toolUseState),
	}
}

// Start は tool_use ブロック開始時の ID/Name を記録する。
func (c *ToolUseCollector) Start(index int, id, name string) {
	c.toolUses[index] = &toolUseState{
		id:   id,
		name: name,
	}
}

// AppendInputDelta は input_json_delta を蓄積する。
func (c *ToolUseCollector) AppendInputDelta(index int, partialJSON string, onInput func(toolName string)) {
	acc := c.toolUses[index]
	if acc == nil || partialJSON == "" {
		return
	}
	if onInput != nil {
		onInput(acc.name)
	}
	acc.input.WriteString(partialJSON)
}

// StopAndEncode は完了した tool_use を内部 JSON 形式へ変換する。
// 変換失敗時は空文字を返す。
func (c *ToolUseCollector) StopAndEncode(index int, encode func(id, name string, input map[string]interface{}) (string, error)) string {
	acc := c.toolUses[index]
	if acc == nil || encode == nil {
		return ""
	}
	delete(c.toolUses, index)

	var input map[string]interface{}
	if err := json.Unmarshal([]byte(acc.input.String()), &input); err != nil {
		return ""
	}
	toolJSON, err := encode(acc.id, acc.name, input)
	if err != nil {
		return ""
	}
	return toolJSON
}

// CompactionCollector は compaction ブロックを index ごとに蓄積する。
type CompactionCollector struct {
	blocks map[int]*strings.Builder
	output strings.Builder
}

// NewCompactionCollector は空の compaction collector を返す。
func NewCompactionCollector() *CompactionCollector {
	return &CompactionCollector{
		blocks: make(map[int]*strings.Builder),
	}
}

// Start は compaction ブロック開始を記録する。
func (c *CompactionCollector) Start(index int) {
	c.blocks[index] = &strings.Builder{}
}

// AppendText は compaction ブロック内テキストを蓄積する。
// 対象 index が compaction ブロックなら true を返す。
func (c *CompactionCollector) AppendText(index int, text string) bool {
	acc, ok := c.blocks[index]
	if !ok {
		return false
	}
	acc.WriteString(text)
	return true
}

// Stop は compaction ブロックを確定し、最終出力へマージする。
func (c *CompactionCollector) Stop(index int) {
	acc, ok := c.blocks[index]
	if !ok {
		return
	}
	c.output.WriteString(acc.String())
	delete(c.blocks, index)
}

// Output は確定済み compaction テキストを返す。
func (c *CompactionCollector) Output() string {
	return c.output.String()
}

// ContentBlockCollector は assistant content blocks を Anthropic の index 順で蓄積する。
type ContentBlockCollector struct {
	blocks    map[int]*contentBlockState
	finalized map[int]api.AnthropicContentBlock
}

type contentBlockState struct {
	block api.AnthropicContentBlock
	input strings.Builder
}

// NewContentBlockCollector は空の content block collector を返す。
func NewContentBlockCollector() *ContentBlockCollector {
	return &ContentBlockCollector{
		blocks:    make(map[int]*contentBlockState),
		finalized: make(map[int]api.AnthropicContentBlock),
	}
}

// Start は content block の開始を記録する。
func (c *ContentBlockCollector) Start(index int, block *ContentBlock) bool {
	if c == nil || block == nil {
		return false
	}

	state := &contentBlockState{}
	switch block.Type {
	case "text":
		state.block = api.AnthropicContentBlock{Type: "text", Text: block.Text}
	case "thinking":
		state.block = api.AnthropicContentBlock{
			Type:      "thinking",
			Thinking:  block.Thinking,
			Signature: block.Signature,
		}
	case "redacted_thinking":
		state.block = api.AnthropicContentBlock{Type: "redacted_thinking", Data: block.Data}
	case "tool_use":
		state.block = api.AnthropicContentBlock{
			Type:  "tool_use",
			ID:    block.ID,
			Name:  block.Name,
			Input: cloneInterfaceMap(block.Input),
		}
	case "compaction":
		state.block = api.AnthropicContentBlock{Type: "compaction"}
	default:
		return false
	}

	c.blocks[index] = state
	return true
}

// AppendDelta は content block delta を蓄積する。
func (c *ContentBlockCollector) AppendDelta(index int, delta *Delta) bool {
	if c == nil || delta == nil {
		return false
	}
	state := c.blocks[index]
	if state == nil {
		return false
	}

	switch delta.Type {
	case "text_delta":
		if state.block.Type == "compaction" {
			state.block.Content += delta.Text
			return true
		}
		state.block.Text += delta.Text
	case "thinking_delta":
		state.block.Thinking += delta.Thinking
	case "signature_delta":
		state.block.Signature = delta.Signature
	case "redacted_thinking_delta":
		state.block.Data += delta.Data
	case "input_json_delta":
		state.input.WriteString(delta.PartialJSON)
	default:
		return false
	}
	return true
}

// Stop は content block を確定する。
func (c *ContentBlockCollector) Stop(index int) bool {
	if c == nil {
		return false
	}
	state := c.blocks[index]
	if state == nil {
		return false
	}
	block := state.block
	if block.Type == "tool_use" && state.input.Len() > 0 {
		var input map[string]any
		if err := json.Unmarshal([]byte(state.input.String()), &input); err == nil {
			block.Input = input
		}
		if block.Input == nil {
			block.Input = map[string]any{}
		}
	}
	c.finalized[index] = block
	delete(c.blocks, index)
	return true
}

// Blocks は確定済み content blocks を Anthropic index 順に返す。
func (c *ContentBlockCollector) Blocks() []api.AnthropicContentBlock {
	if c == nil || len(c.finalized) == 0 {
		return nil
	}
	indices := make([]int, 0, len(c.finalized))
	for index := range c.finalized {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	out := make([]api.AnthropicContentBlock, 0, len(indices))
	for _, index := range indices {
		out = append(out, c.finalized[index])
	}
	return out
}

func cloneInterfaceMap(src map[string]interface{}) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

package claudestream

import (
	"encoding/json"
	"strings"
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

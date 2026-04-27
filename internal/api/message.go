package api

// Message はチャットメッセージ
type Message struct {
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"` // DeepSeek Reasoner の思考内容
	ToolCallID       string           `json:"tool_call_id,omitempty"`      // Function Calling: ツール結果用
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`        // Function Calling: assistant の tool_calls
	ToolName         string           `json:"tool_name,omitempty"`         // Gemini functionResponse 用ツール名

	providerState messageProviderState
}

type messageProviderState struct {
	anthropicContentBlocks  []AnthropicContentBlock
	anthropicThinkingBlocks []AnthropicThinkingBlock
}

// AnthropicThinkingBlock は Claude extended thinking の再送が必要なブロックを表す。
type AnthropicThinkingBlock struct {
	Type      string `json:"type"`                // "thinking" or "redacted_thinking"
	Thinking  string `json:"thinking,omitempty"`  // type="thinking" 用
	Signature string `json:"signature,omitempty"` // type="thinking" 用
	Data      string `json:"data,omitempty"`      // type="redacted_thinking" 用
}

// AnthropicContentBlock は Claude thinking/tool-use 継続用の provider 専用 content block を表す。
type AnthropicContentBlock struct {
	Type      string         `json:"type"`                // "text", "thinking", "redacted_thinking", "tool_use", "compaction"
	Text      string         `json:"text,omitempty"`      // type="text" 用
	Thinking  string         `json:"thinking,omitempty"`  // type="thinking" 用
	Signature string         `json:"signature,omitempty"` // type="thinking" 用
	Data      string         `json:"data,omitempty"`      // type="redacted_thinking" 用
	ID        string         `json:"id,omitempty"`        // type="tool_use" 用
	Name      string         `json:"name,omitempty"`      // type="tool_use" 用
	Input     map[string]any `json:"input,omitempty"`     // type="tool_use" 用
	Content   string         `json:"content,omitempty"`   // type="compaction" 用
}

// AnthropicContentBlocks は Claude thinking/tool-use 継続用の ordered provider state を返す。
func (m Message) AnthropicContentBlocks() []AnthropicContentBlock {
	return cloneAnthropicContentBlocks(m.providerState.anthropicContentBlocks)
}

// SetAnthropicContentBlocks は Claude thinking/tool-use 継続用の ordered provider state を設定する。
func (m *Message) SetAnthropicContentBlocks(blocks []AnthropicContentBlock) {
	m.providerState.anthropicContentBlocks = cloneAnthropicContentBlocks(blocks)
	m.providerState.anthropicThinkingBlocks = nil
}

// AnthropicThinkingBlocks は Claude thinking 継続用の provider 専用 state を返す。
func (m Message) AnthropicThinkingBlocks() []AnthropicThinkingBlock {
	if len(m.providerState.anthropicThinkingBlocks) == 0 {
		return AnthropicThinkingBlocksFromContentBlocks(m.providerState.anthropicContentBlocks)
	}
	out := make([]AnthropicThinkingBlock, len(m.providerState.anthropicThinkingBlocks))
	copy(out, m.providerState.anthropicThinkingBlocks)
	return out
}

// SetAnthropicThinkingBlocks は Claude thinking 継続用の provider 専用 state を設定する。
func (m *Message) SetAnthropicThinkingBlocks(blocks []AnthropicThinkingBlock) {
	if len(blocks) == 0 {
		m.providerState.anthropicThinkingBlocks = nil
		m.providerState.anthropicContentBlocks = nil
		return
	}
	m.providerState.anthropicThinkingBlocks = make([]AnthropicThinkingBlock, len(blocks))
	copy(m.providerState.anthropicThinkingBlocks, blocks)
	m.providerState.anthropicContentBlocks = nil
}

// AnthropicThinkingBlocksFromContentBlocks は ordered content blocks から thinking 系 block だけを抽出する。
func AnthropicThinkingBlocksFromContentBlocks(blocks []AnthropicContentBlock) []AnthropicThinkingBlock {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]AnthropicThinkingBlock, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "thinking":
			out = append(out, AnthropicThinkingBlock{
				Type:      "thinking",
				Thinking:  block.Thinking,
				Signature: block.Signature,
			})
		case "redacted_thinking":
			out = append(out, AnthropicThinkingBlock{
				Type: "redacted_thinking",
				Data: block.Data,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneAnthropicContentBlocks(blocks []AnthropicContentBlock) []AnthropicContentBlock {
	return CloneAnthropicContentBlocks(blocks)
}

// CloneAnthropicContentBlocks は ordered content blocks を defensive copy する。
func CloneAnthropicContentBlocks(blocks []AnthropicContentBlock) []AnthropicContentBlock {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]AnthropicContentBlock, len(blocks))
	for i, block := range blocks {
		out[i] = block
		out[i].Input = cloneAnyMap(block.Input)
	}
	return out
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneAnyValue(v)
	}
	return dst
}

func cloneAnyValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneAnyMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneAnyValue(item)
		}
		return out
	default:
		return v
	}
}

// MultimodalMessage は画像を含むメッセージ
type MultimodalMessage struct {
	Role    string     `json:"role"`
	Content string     `json:"content"`
	Image   *ImageData `json:"-"` // JSON保存しない（一時的なもの）
}

// ToMessage は通常のMessageに変換（画像なし）
func (m MultimodalMessage) ToMessage() Message {
	return Message{Role: m.Role, Content: m.Content}
}

// HasImage は画像が添付されているか
func (m MultimodalMessage) HasImage() bool {
	return m.Image != nil && m.Image.Base64 != ""
}

// GetReasoningContent はプロバイダーから最後の reasoning_content を取得するヘルパー
// プロバイダーが ReasoningContentProvider を実装していない場合は空文字を返す
func GetReasoningContent(provider Provider) string {
	if rcp, ok := provider.(ReasoningContentProvider); ok {
		return rcp.LastReasoningContent()
	}
	return ""
}

// GetAnthropicThinkingBlocks はプロバイダーから最後の Claude thinking blocks を取得する。
func GetAnthropicThinkingBlocks(provider Provider) []AnthropicThinkingBlock {
	if provider == nil {
		return nil
	}
	if cp, ok := provider.(AnthropicContentBlockProvider); ok {
		blocks := AnthropicThinkingBlocksFromContentBlocks(cp.LastAnthropicContentBlocks())
		if len(blocks) == 0 {
			return nil
		}
		return blocks
	}
	if tp, ok := provider.(AnthropicThinkingBlockProvider); ok {
		blocks := tp.LastAnthropicThinkingBlocks()
		if len(blocks) == 0 {
			return nil
		}
		out := make([]AnthropicThinkingBlock, len(blocks))
		copy(out, blocks)
		return out
	}
	return nil
}

// GetAnthropicContentBlocks はプロバイダーから最後の Claude ordered content blocks を取得する。
func GetAnthropicContentBlocks(provider Provider) []AnthropicContentBlock {
	if provider == nil {
		return nil
	}
	if cp, ok := provider.(AnthropicContentBlockProvider); ok {
		return cloneAnthropicContentBlocks(cp.LastAnthropicContentBlocks())
	}
	if tp, ok := provider.(AnthropicThinkingBlockProvider); ok {
		thinkingBlocks := tp.LastAnthropicThinkingBlocks()
		if len(thinkingBlocks) == 0 {
			return nil
		}
		out := make([]AnthropicContentBlock, 0, len(thinkingBlocks))
		for _, block := range thinkingBlocks {
			switch block.Type {
			case "thinking":
				out = append(out, AnthropicContentBlock{
					Type:      "thinking",
					Thinking:  block.Thinking,
					Signature: block.Signature,
				})
			case "redacted_thinking":
				out = append(out, AnthropicContentBlock{
					Type: "redacted_thinking",
					Data: block.Data,
				})
			}
		}
		return out
	}
	return nil
}

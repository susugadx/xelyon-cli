package api

import "strings"

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
	openAIResponsesItems    []InputItem
	image                   *ImageData
}

// AnthropicThinkingBlock は Claude extended thinking の再送が必要なブロックを表す。
type AnthropicThinkingBlock struct {
	Type      string `json:"type"`                // "thinking" or "redacted_thinking"
	Thinking  string `json:"thinking,omitempty"`  // type="thinking" 用
	Signature string `json:"signature,omitempty"` // type="thinking" 用
	Data      string `json:"data,omitempty"`      // type="redacted_thinking" 用
}

// ImageSource は provider-facing image block の source shape を表す。
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png" など
	Data      string `json:"data"`       // base64
}

// AnthropicContentBlock は Claude thinking/tool-use 継続用の provider 専用 content block を表す。
type AnthropicContentBlock struct {
	Type      string         `json:"type"`                // "text", "thinking", "redacted_thinking", "tool_use", "compaction"
	Text      string         `json:"text,omitempty"`      // type="text" 用
	Source    *ImageSource   `json:"source,omitempty"`    // type="image" 用
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

// OpenAIResponsesInputItems は Responses full-payload replay 用の provider-facing items を返す。
func (m Message) OpenAIResponsesInputItems() []InputItem {
	return CloneInputItems(m.providerState.openAIResponsesItems)
}

// SetOpenAIResponsesInputItems は Responses full-payload replay 用の provider-facing items を設定する。
func (m *Message) SetOpenAIResponsesInputItems(items []InputItem) {
	m.providerState.openAIResponsesItems = CloneInputItems(items)
}

// NewUserImageMessage は画像付き user message を runtime-only state として保持する。
func NewUserImageMessage(content string, image *ImageData) Message {
	msg := NewUserMessage(content)
	msg.SetImageData(image)
	return msg
}

// NewUserMessage は text-only user message を作成する。
func NewUserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// NewUserMessageWithOptionalImage は画像があれば runtime-only state に保持した user message を作成する。
func NewUserMessageWithOptionalImage(content string, image *ImageData) Message {
	if validImageData(image) {
		return NewUserImageMessage(content, image)
	}
	return NewUserMessage(content)
}

// HasImage は runtime-only 画像 state を持つかを返す。
func (m Message) HasImage() bool {
	return validImageData(m.providerState.image)
}

// ImageData は runtime-only 画像 state を defensive copy して返す。
func (m Message) ImageData() *ImageData {
	return cloneImageData(m.providerState.image)
}

// SetImageData は runtime-only 画像 state を設定する。
func (m *Message) SetImageData(image *ImageData) {
	m.providerState.image = cloneImageData(image)
}

// MessagesHaveImage は履歴に画像付き message が含まれるかを返す。
func MessagesHaveImage(messages []Message) bool {
	for _, msg := range messages {
		if msg.HasImage() {
			return true
		}
	}
	return false
}

// ReplaceOpenAIResponsesFunctionCallArguments は replay metadata 内の function_call arguments を同期する。
func (m *Message) ReplaceOpenAIResponsesFunctionCallArguments(callID, name, arguments string) bool {
	items := m.OpenAIResponsesInputItems()
	if len(items) == 0 {
		return true
	}

	hasFunctionCall := false
	updated := false
	for i := range items {
		if items[i].Type != "function_call" {
			continue
		}
		hasFunctionCall = true
		if items[i].CallID != callID {
			continue
		}
		if items[i].Name != "" && items[i].Name != name {
			return false
		}
		items[i].Name = name
		items[i].Arguments = arguments
		updated = true
	}
	if hasFunctionCall && !updated {
		return false
	}
	m.SetOpenAIResponsesInputItems(items)
	return true
}

// ReplaceOpenAIResponsesFunctionCallOutput は replay metadata 内の function_call_output を同期する。
func (m *Message) ReplaceOpenAIResponsesFunctionCallOutput(callID, output string) {
	items := m.OpenAIResponsesInputItems()
	if len(items) == 0 {
		return
	}

	updated := false
	for i := range items {
		if items[i].Type != "function_call_output" || items[i].CallID != callID {
			continue
		}
		items[i].Output = output
		items[i] = NormalizeInputItemOutput(items[i])
		updated = true
	}
	if updated {
		m.SetOpenAIResponsesInputItems(items)
	}
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
		if block.Source != nil {
			source := *block.Source
			out[i].Source = &source
		}
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

// ToMessage は画像 state を保持した Message に変換する。
func (m MultimodalMessage) ToMessage() Message {
	msg := Message{Role: m.Role, Content: m.Content}
	msg.SetImageData(m.Image)
	return msg
}

// HasImage は画像が添付されているか
func (m MultimodalMessage) HasImage() bool {
	return validImageData(m.Image)
}

func validImageData(image *ImageData) bool {
	return image != nil && strings.TrimSpace(image.Base64) != ""
}

func cloneImageData(image *ImageData) *ImageData {
	if image == nil {
		return nil
	}
	cloned := *image
	return &cloned
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

// GetOpenAIResponsesInputItems は provider から最後の Responses replay items を取得する。
func GetOpenAIResponsesInputItems(provider Provider) []InputItem {
	if provider == nil {
		return nil
	}
	if rp, ok := provider.(OpenAIResponsesReplayProvider); ok {
		return CloneInputItems(rp.LastOpenAIResponsesInputItems())
	}
	return nil
}

package api

// Message はチャットメッセージ
type Message struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // Function Calling: ツール結果用
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`   // Function Calling: assistant の tool_calls
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

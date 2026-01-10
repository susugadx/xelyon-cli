package api

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

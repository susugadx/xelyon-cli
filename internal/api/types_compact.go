package api

// このファイルは OpenAI Compact API / Responses API で使用される型を定義します。
// CompactCapable インターフェースで使用されるため、api パッケージに配置されています。

// InputItem は Responses API の入力アイテム（Compact API対応拡張版）
// ユーザーメッセージ、アシスタント応答、圧縮済みアイテムを表現
type InputItem struct {
	Type    string      `json:"type"`              // "message" or "compacted"
	Role    string      `json:"role,omitempty"`    // "user", "assistant"
	Content interface{} `json:"content,omitempty"` // string or []InputContentPart

	// アシスタント応答の完全情報（Compact API用）
	ID     string `json:"id,omitempty"`     // "msg_xxx" (アシスタント応答のID)
	Status string `json:"status,omitempty"` // "completed"

	// 圧縮済みアイテム用
	Data string `json:"data,omitempty"` // 暗号化データ（type="compacted"の場合）
}

// InputContentPart は Responses API のコンテンツパート（画像対応）
type InputContentPart struct {
	Type     string `json:"type"`                // "input_text" or "input_image"
	Text     string `json:"text,omitempty"`      // type="input_text"の場合
	ImageURL string `json:"image_url,omitempty"` // type="input_image"の場合（data:image/...形式）
}

// CompactResponse は /responses/compact レスポンス
type CompactResponse struct {
	Output []InputItem   `json:"output"` // 圧縮済みアイテム（次の /responses に使用）
	Model  string        `json:"model"`
	Usage  *CompactUsage `json:"usage,omitempty"`
}

// CompactUsage はトークン使用量
type CompactUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ConvertHistoryToInputItems は History を InputItem 形式に変換
// Compact API に送信するためのフル会話ウィンドウを構築
func ConvertHistoryToInputItems(history []Message) []InputItem {
	items := make([]InputItem, 0, len(history))
	for _, msg := range history {
		items = append(items, InputItem{
			Type:    "message",
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return items
}

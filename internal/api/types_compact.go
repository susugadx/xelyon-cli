package api

// このファイルは OpenAI Compact API / Responses API で使用される型を定義します。
// CompactCapable インターフェースで使用されるため、api パッケージに配置されています。

// InputItem は Responses API の入力アイテム（Compact API対応拡張版）
// ユーザーメッセージ、アシスタント応答、圧縮済みアイテム、Function Calling を表現
type InputItem struct {
	Type    string      `json:"type"`              // "message", "compacted", "function_call", "function_call_output"
	Role    string      `json:"role,omitempty"`    // "user", "assistant"
	Content interface{} `json:"content,omitempty"` // string or []InputContentPart

	// アシスタント応答の完全情報（Compact API用）
	ID     string `json:"id,omitempty"`     // "msg_xxx" (アシスタント応答のID)
	Status string `json:"status,omitempty"` // "completed"

	// 圧縮済みアイテム用
	Data string `json:"data,omitempty"` // 暗号化データ（type="compacted"の場合）

	// Function Calling 用（type="function_call"の場合）
	CallID    string `json:"call_id,omitempty"`   // ツール呼び出しID
	Name      string `json:"name,omitempty"`      // ツール名
	Arguments string `json:"arguments,omitempty"` // 引数（JSON文字列）

	// Function Calling 結果用（type="function_call_output"の場合）
	Output string `json:"output,omitempty"` // ツール実行結果
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
// Function Calling: assistant+ToolCalls は "function_call"、role="tool" は "function_call_output" 形式に変換
func ConvertHistoryToInputItems(history []Message) []InputItem {
	items := make([]InputItem, 0, len(history))
	for _, msg := range history {
		if msg.Role == "tool" {
			// Function Calling 結果は function_call_output 形式
			items = append(items, InputItem{
				Type:   "function_call_output",
				CallID: msg.ToolCallID,
				Output: msg.Content,
			})
		} else if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// AIのfunction_call → type: "function_call" として各ツール呼び出しを追加
			for _, tc := range msg.ToolCalls {
				items = append(items, InputItem{
					Type:      "function_call",
					CallID:    tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		} else {
			items = append(items, InputItem{
				Type:    "message",
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	return items
}

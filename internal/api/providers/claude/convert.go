package claude

import (
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// AnthropicContentBlock は Anthropic Messages API のコンテンツブロック
// text, tool_use, tool_result の3種類を統一的に表現
type AnthropicContentBlock struct {
	Type         string            `json:"type"`                    // "text", "tool_use", "tool_result"
	Text         string            `json:"text,omitempty"`          // type="text" 用
	ID           string            `json:"id,omitempty"`            // type="tool_use" 用
	Name         string            `json:"name,omitempty"`          // type="tool_use" 用
	Input        map[string]any    `json:"input,omitempty"`         // type="tool_use" 用
	ToolUseID    string            `json:"tool_use_id,omitempty"`   // type="tool_result" 用
	Content      string            `json:"content,omitempty"`       // type="tool_result" / "compaction" 用
	CacheControl *api.CacheControl `json:"cache_control,omitempty"` // プロンプトキャッシュ用
}

// AnthropicMessage は Anthropic Messages API 形式のメッセージ
// Content は常に []AnthropicContentBlock 配列
type AnthropicMessage struct {
	Role    string                  `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

// ConvertToAnthropicMessages は api.Message のスライスを Anthropic Messages API 形式に変換する。
//
// 変換ルール:
//   - role:"user" → role:"user" + [{type:"text", text:...}]
//   - role:"assistant" (ToolCalls なし) → role:"assistant" + [{type:"text", text:...}]
//   - role:"assistant" + ToolCalls → role:"assistant" + [{type:"tool_use",...}] (テキストがあれば先頭に text ブロック追加)
//   - role:"tool" → role:"user" + [{type:"tool_result", tool_use_id:..., content:...}]
//   - 連続する同一 role のメッセージはマージ（Anthropic API は user/assistant 交互を要求）
func ConvertToAnthropicMessages(history []api.Message) []AnthropicMessage {
	if len(history) == 0 {
		return nil
	}

	var result []AnthropicMessage

	for _, msg := range history {
		var converted AnthropicMessage

		switch {
		case msg.Role == "tool":
			// tool → user + tool_result ブロック
			converted = AnthropicMessage{
				Role: "user",
				Content: []AnthropicContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   msg.Content,
					},
				},
			}

		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			// assistant + ToolCalls → assistant + tool_use ブロック
			var blocks []AnthropicContentBlock

			// テキスト部分があれば先頭に追加
			if msg.Content != "" {
				blocks = append(blocks, AnthropicContentBlock{
					Type: "text",
					Text: msg.Content,
				})
			}

			// 各 ToolCall を tool_use ブロックに変換
			for _, tc := range msg.ToolCalls {
				block := AnthropicContentBlock{
					Type: "tool_use",
					ID:   tc.ID,
					Name: tc.Function.Name,
				}

				// Arguments (JSON文字列) を map[string]any にパース
				var input map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err == nil {
					block.Input = input
				} else {
					// パース失敗時は空の input
					block.Input = map[string]any{}
				}

				blocks = append(blocks, block)
			}

			converted = AnthropicMessage{
				Role:    "assistant",
				Content: blocks,
			}

		default:
			// 通常の user/assistant メッセージ
			// assistant メッセージに compaction が含まれている場合、構造化して変換
			if msg.Role == "assistant" && strings.Contains(msg.Content, "[COMPACTION]") {
				compactionSummary, textContent := extractCompaction(msg.Content)
				var blocks []AnthropicContentBlock

				if compactionSummary != "" {
					blocks = append(blocks, AnthropicContentBlock{
						Type:    "compaction",
						Content: compactionSummary,
					})
				}

				if textContent != "" {
					blocks = append(blocks, AnthropicContentBlock{
						Type: "text",
						Text: textContent,
					})
				}

				converted = AnthropicMessage{
					Role:    "assistant",
					Content: blocks,
				}
			} else {
				textContent := msg.Content
				if textContent == "" {
					textContent = "(empty)"
				}
				converted = AnthropicMessage{
					Role: msg.Role,
					Content: []AnthropicContentBlock{
						{
							Type: "text",
							Text: textContent,
						},
					},
				}
			}
		}

		// 連続する同一 role のメッセージはマージ
		if len(result) > 0 && result[len(result)-1].Role == converted.Role {
			result[len(result)-1].Content = append(result[len(result)-1].Content, converted.Content...)
		} else {
			result = append(result, converted)
		}
	}

	return result
}

// SetMessageCacheBreakpoints はメッセージ履歴にキャッシュ用ブレークポイントを設定する。
//
// BP配置戦略:
//
//	BP#3 のみ使用。BP#4 は使わない（毎ターンcreation 1.25xが発生し、1度もREADされないため）。
//
//	BP#3: 安定区間の末尾userメッセージ。
//	位置は bpInterval 単位で切り捨てて決定するため、interval ターンの間は同じ位置に留まる。
//	これにより cache creation は interval ターンに1回だけ発生し、残りは cache read になる。
//
// 計算方法:
//  1. userメッセージ数から stableOffset を引く = 安定区間のuser数
//  2. bpInterval の倍数に切り捨て = BP#3の位置（userインデックス）
//  3. 安定区間のuser数が bpInterval 未満なら BP#3 は設定しない
//
// stableOffset=3: 最新3ターン分をキャッシュ外に置く（CompactOldToolResults の keepTurns=3 と一致）
// bpInterval=3: 3ターンごとにBP#3が進む（creation頻度とカバー範囲のバランス）
func SetMessageCacheBreakpoints(messages []AnthropicMessage) {
	const stableOffset = 3
	const bpInterval = 3

	// user メッセージのインデックスを収集
	var userIndices []int
	for i, msg := range messages {
		if msg.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}

	if len(userIndices) == 0 {
		return
	}

	// 安定区間の user 数
	stableCount := len(userIndices) - stableOffset
	if stableCount < bpInterval {
		return // 短い会話では BP#3 不要（BP#1, #2 で十分）
	}

	// bpInterval の倍数に切り捨て → BP#3 の位置
	roundedDown := (stableCount / bpInterval) * bpInterval
	if roundedDown <= 0 {
		return
	}

	// BP#3: 安定区間末尾の user メッセージ
	bp3Idx := userIndices[roundedDown-1]
	if len(messages[bp3Idx].Content) > 0 {
		last := len(messages[bp3Idx].Content) - 1
		messages[bp3Idx].Content[last].CacheControl = &api.CacheControl{Type: "ephemeral"}
	}
}

// extractCompaction は [COMPACTION] マーカーからサマリーと残りのテキストを抽出する
func extractCompaction(content string) (string, string) {
	const startMarker = "[COMPACTION]"
	const endMarker = "[/COMPACTION]"

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return "", content
	}

	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 {
		return "", content
	}

	compactionSummary := strings.TrimSpace(content[startIdx+len(startMarker) : endIdx])
	textContent := strings.TrimSpace(content[:startIdx] + content[endIdx+len(endMarker):])

	return compactionSummary, textContent
}

package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
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

// SetMessageCacheBreakpointsWithEnabled は prompt cache 有効時のみメッセージに breakpoints を設定する。
func SetMessageCacheBreakpointsWithEnabled(messages []AnthropicMessage, enabled bool) {
	SetMessageCacheBreakpointsWithConfigAndEnabled(messages, config.DefaultConfig(), enabled)
}

// SetMessageCacheBreakpointsWithConfigAndEnabled は設定と有効フラグを指定して breakpoints を設定する。
func SetMessageCacheBreakpointsWithConfigAndEnabled(messages []AnthropicMessage, cfg *config.Config, enabled bool) {
	if !enabled {
		return
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	const stableOffset = 3

	// user メッセージのインデックスを収集
	var userIndices []int
	for i, msg := range messages {
		if msg.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}

	if len(userIndices) <= stableOffset {
		return // 短い会話では message BP 不要（BP#1, #2 で十分）
	}

	// 安定区間の境界: この index より前のメッセージのみ対象
	boundaryIdx := userIndices[len(userIndices)-stableOffset]

	// 安定区間内の tool_result ブロックを content 長で収集
	type candidate struct {
		msgIdx   int // messages 配列のインデックス
		blockIdx int // Content 配列のインデックス
		tokens   int // content の概算トークン数
	}
	var candidates []candidate

	for i := 0; i < boundaryIdx; i++ {
		if messages[i].Role != "user" {
			continue
		}
		for j, block := range messages[i].Content {
			if block.Type == "tool_result" && len(block.Content) > 0 {
				candidates = append(candidates, candidate{
					msgIdx:   i,
					blockIdx: j,
					tokens:   token.EstimateTokenCount(block.Content),
				})
			}
		}
	}

	if len(candidates) == 0 {
		return
	}

	// content 長で降順ソート
	sort.Slice(candidates, func(a, b int) bool {
		return candidates[a].tokens > candidates[b].tokens
	})

	// 上位2つに cache_control を設定（BP#3, BP#4）
	limit := 2
	if len(candidates) < limit {
		limit = len(candidates)
	}
	for i := 0; i < limit; i++ {
		c := candidates[i]
		messages[c.msgIdx].Content[c.blockIdx].CacheControl = api.NewCacheControlWithConfig(cfg)
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

// validateAnthropicToolPairs は変換後のメッセージで tool_use/tool_result の整合性を検証し、
// 不整合がある場合にデバッグ情報を出力する。
func validateAnthropicToolPairs(messages []AnthropicMessage, out io.Writer) {
	for i, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		// この user メッセージに tool_result があるか確認
		var toolResultIDs []string
		for _, block := range msg.Content {
			if block.Type == "tool_result" {
				toolResultIDs = append(toolResultIDs, block.ToolUseID)
			}
		}
		if len(toolResultIDs) == 0 {
			continue
		}

		// 直前の assistant メッセージの tool_use IDs を収集
		var toolUseIDs []string
		if i > 0 && messages[i-1].Role == "assistant" {
			for _, block := range messages[i-1].Content {
				if block.Type == "tool_use" {
					toolUseIDs = append(toolUseIDs, block.ID)
				}
			}
		}

		// tool_result の ID が tool_use に全て含まれているか
		useIDSet := make(map[string]bool)
		for _, id := range toolUseIDs {
			useIDSet[id] = true
		}
		for j, resultID := range toolResultIDs {
			if !useIDSet[resultID] {
				fmt.Fprintf(out, "[DEBUG Claude] ❌ messages[%d].content[%d]: tool_result.tool_use_id=%q NOT in preceding assistant tool_use IDs %v\n",
					i, j, resultID, toolUseIDs)
			}
		}

		// 数の不一致もチェック
		if len(toolResultIDs) != len(toolUseIDs) {
			fmt.Fprintf(out, "[DEBUG Claude] ⚠️  messages[%d]: %d tool_results vs %d tool_uses in messages[%d]\n",
				i, len(toolResultIDs), len(toolUseIDs), i-1)
		}
	}
}

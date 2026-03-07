package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	DefaultKeepTurns = 3  // 最新3ターンはそのまま
	DefaultMaxLines  = 50 // 50行以下はtruncateしない
	DefaultHeadLines = 20 // 先頭20行を保持
	DefaultTailLines = 5  // 末尾5行を保持
)

// CompactOldToolResults は送信前に古いツール結果をtruncateした履歴コピーを返す。
// 元の history は変更しない（セッション保存用に原本を保持）。
//
// ルール:
// - "user" メッセージの出現回数で「ターン」をカウント（assistantのtool呼び出し→tool結果は同一ターン）
// - 最新 keepTurns ターン以内の tool 結果はそのまま
// - それより古い tool 結果は段階的に truncate（古いほど短く）
//
// 段階的圧縮:
// - keepTurns+1 〜 keepTurns*2: headLines/tailLines（デフォルト 20/5）
// - keepTurns*2+1 〜 keepTurns*3: headLines=10, tailLines=3
// - それ以上古い: headLines=5, tailLines=2
//
// 失敗ペア圧縮:
//   - keepTurns より古いターンで、tool result が "Error:" で始まり、
//     直後の assistant が同じツールを再呼び出しする場合、
//     失敗 tool result を 1行サマリーに圧縮
//
// 注意: shallow copy のため、返されたスライスの Content 以外のフィールド（ToolCalls 等）は
// 元の history と共有されている。返り値の Content 以外を変更してはならない。
func CompactOldToolResults(history []api.Message, keepTurns, maxLines, headLines, tailLines int) []api.Message {
	if len(history) == 0 {
		return history
	}

	// ターン境界を計算: 末尾から user メッセージを数えて keepTurns 個目の位置を求める
	boundary := 0 // この index より前のメッセージが「古い」
	userCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			userCount++
			if userCount >= keepTurns {
				boundary = i
				break
			}
		}
	}

	// keepTurns 未満のターンしかない場合はコピーをそのまま返す
	if boundary == 0 {
		result := make([]api.Message, len(history))
		copy(result, history)
		return result
	}

	// ターン境界マップ: 各 index が boundary からどれだけ古いか（ターン数）
	// boundary から逆方向に user メッセージを数える
	turnAge := buildTurnAgeMap(history, boundary)

	// shallow copy + 古い tool 結果のみ段階的 truncate
	result := make([]api.Message, len(history))
	copy(result, history)

	for i := 0; i < boundary; i++ {
		if result[i].Role == "tool" {
			// 失敗ペア圧縮: Error で始まり、直後の assistant が同じツールを再呼び出し
			if compressed, ok := compressFailedPair(result, i); ok {
				result[i] = compressed
				continue
			}

			// 段階的圧縮: ターンの古さに応じて truncate 量を変える
			age := turnAge[i]
			h, t := graduatedTruncateParams(age, keepTurns, headLines, tailLines)
			result[i] = truncateToolResult(result[i], maxLines, h, t)
		}
	}

	return result
}

// buildTurnAgeMap は boundary より前の各 index について、
// boundary からのターン距離（user メッセージ数ベース）を返す。
// boundary 直前のターンが age=1、その前が age=2、…。
func buildTurnAgeMap(history []api.Message, boundary int) map[int]int {
	ages := make(map[int]int)
	age := 0
	// boundary から逆方向にスキャン
	for i := boundary - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			age++
		}
		ages[i] = age
	}
	return ages
}

// graduatedTruncateParams はターンの古さに応じた truncate パラメータを返す。
func graduatedTruncateParams(age, keepTurns, defaultHead, defaultTail int) (headLines, tailLines int) {
	switch {
	case age <= keepTurns:
		// keepTurns+1 〜 keepTurns*2 相当（boundary直前の keepTurns ターン）
		return defaultHead, defaultTail
	case age <= keepTurns*2:
		// keepTurns*2+1 〜 keepTurns*3 相当
		return 10, 3
	default:
		// それ以上古い
		return 5, 2
	}
}

// compressFailedPair は失敗ペアを検出して圧縮する。
// tool result が "Error:" で始まり、直後の assistant メッセージが同じツールを再呼び出しする場合、
// 1行サマリーに圧縮した Message を返す。
func compressFailedPair(history []api.Message, toolIdx int) (api.Message, bool) {
	content := strings.TrimSpace(history[toolIdx].Content)
	if !strings.HasPrefix(content, "Error:") {
		return api.Message{}, false
	}

	// 直後の assistant メッセージを探す
	assistantIdx := -1
	for j := toolIdx + 1; j < len(history); j++ {
		if history[j].Role == "assistant" {
			assistantIdx = j
			break
		}
		// user が先に来たら失敗ペアではない
		if history[j].Role == "user" {
			return api.Message{}, false
		}
	}
	if assistantIdx < 0 {
		return api.Message{}, false
	}

	// assistant が同じツールを再呼び出ししているかチェック
	failedToolName := history[toolIdx].ToolName
	if failedToolName == "" {
		return api.Message{}, false
	}
	retried := false
	for _, tc := range history[assistantIdx].ToolCalls {
		if tc.Function.Name == failedToolName {
			retried = true
			break
		}
	}
	if !retried {
		return api.Message{}, false
	}

	// 1行サマリーに圧縮
	firstLine := content
	if idx := strings.IndexByte(content, '\n'); idx >= 0 {
		firstLine = content[:idx]
	}
	compressed := history[toolIdx]
	compressed.Content = fmt.Sprintf("[Failed: %s — %s]", failedToolName, firstLine)
	return compressed, true
}

// compressErrorResult はエラーパターンを検出して短い要約に圧縮する。
// 圧縮した場合は (要約, true) を返す。エラーパターンでなければ ("", false)。
func compressErrorResult(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)

	// "No matches found" → 既に短いのでそのまま
	if trimmed == "No matches found" {
		return trimmed, true
	}

	// "Error: pattern not found" / "Error: old_str not found"
	if strings.HasPrefix(trimmed, "Error: pattern not found") ||
		strings.HasPrefix(trimmed, "Error: old_str not found") {
		// 1行目だけ残す
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	// "Error reading file:" → 1行目だけ
	if strings.HasPrefix(trimmed, "Error reading file:") {
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	// "Error:" で始まる結果全般 → 1行目だけ
	if strings.HasPrefix(trimmed, "Error:") {
		if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
			return trimmed[:idx], true
		}
		return trimmed, true
	}

	return "", false
}

// truncateToolResult は大きなツール結果を truncate する。
// エラーパターンは行数に関係なく圧縮する。
func truncateToolResult(msg api.Message, maxLines, headLines, tailLines int) api.Message {
	// エラーパターンの圧縮を先に試す
	if compressed, ok := compressErrorResult(msg.Content); ok {
		if compressed != msg.Content {
			result := msg
			result.Content = compressed
			return result
		}
		return msg
	}

	lines := strings.Split(strings.TrimRight(msg.Content, "\n"), "\n")
	if len(lines) <= maxLines || headLines+tailLines >= len(lines) {
		return msg
	}

	// おおよそのトークン数を推定（~4文字/トークン）
	estimatedTokens := len(msg.Content) / 4

	head := strings.Join(lines[:headLines], "\n")
	tail := strings.Join(lines[len(lines)-tailLines:], "\n")
	truncated := fmt.Sprintf("%s\n... (truncated: was %d lines, ~%d tokens)\n%s",
		head, len(lines), estimatedTokens, tail)

	// shallow copy して Content だけ差し替え
	result := msg
	result.Content = truncated
	return result
}

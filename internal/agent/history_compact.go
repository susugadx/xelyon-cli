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
// - それより古い tool 結果は truncate（大きい場合のみ）
//
// truncate方式:
// - 結果が maxLines 行を超える場合: 先頭 headLines 行 + "\n... (truncated: was N lines, ~M tokens)" + 末尾 tailLines 行
// - maxLines 以下の場合: そのまま
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

	// shallow copy + 古い tool 結果のみ truncate
	result := make([]api.Message, len(history))
	copy(result, history)

	for i := 0; i < boundary; i++ {
		if result[i].Role == "tool" {
			result[i] = truncateToolResult(result[i], maxLines, headLines, tailLines)
		}
	}

	return result
}

// truncateToolResult は大きなツール結果を truncate する
func truncateToolResult(msg api.Message, maxLines, headLines, tailLines int) api.Message {
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

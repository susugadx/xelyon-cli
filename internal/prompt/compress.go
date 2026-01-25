package prompt

import (
	"fmt"
	"strings"
)

// Message represents a conversation message for summary generation.
// This is a simplified struct to avoid importing api package.
type Message struct {
	Role    string
	Content string
}

// BuildSummaryPrompt generates the prompt for summarizing conversation history.
// truncateLen specifies the maximum length of each message before truncation.
func BuildSummaryPrompt(messages []Message, truncateLen int) string {
	var sb strings.Builder

	sb.WriteString("以下の会話履歴を簡潔なサマリーにまとめてください。\n")
	sb.WriteString("重要な決定事項、作成/編集したファイル、残タスクを含めてください。\n")
	sb.WriteString("箇条書きで5-10項目程度にまとめてください。\n\n")
	sb.WriteString("会話履歴:\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for _, msg := range messages {
		var role string
		switch msg.Role {
		case "assistant":
			role = "Assistant"
		case "system":
			role = "System"
		default:
			role = "User"
		}

		// 長いメッセージは省略
		content := msg.Content
		if len(content) > truncateLen {
			content = content[:truncateLen] + "... (省略)"
		}

		sb.WriteString(fmt.Sprintf("[%s]\n%s\n\n", role, content))
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString("上記の会話を要約してください。日本語で回答してください。")

	return sb.String()
}

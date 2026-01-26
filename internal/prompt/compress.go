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

// BuildSummaryPrompt はサマリー生成用のプロンプトを構築する。
// truncateLen は各メッセージの最大長（超過分は省略）。
func BuildSummaryPrompt(messages []Message, truncateLen int) string {
	var sb strings.Builder

	sb.WriteString("Summarize this conversation history into a concise summary.\n\n")
	sb.WriteString("Include:\n")
	sb.WriteString("- Key decisions made\n")
	sb.WriteString("- Files created/modified\n")
	sb.WriteString("- Remaining tasks (if any)\n\n")
	sb.WriteString("Output as bullet points (5-8 items max).\n")
	sb.WriteString("Respond in the same language as the conversation.\n\n")
	sb.WriteString("---\n\n")

	for _, msg := range messages {
		// systemメッセージはスキップ
		if msg.Role == "system" {
			continue
		}

		var role string
		switch msg.Role {
		case "assistant":
			role = "Assistant"
		default:
			role = "User"
		}

		// 長いメッセージは省略
		content := msg.Content
		if len(content) > truncateLen {
			content = content[:truncateLen] + "..."
		}

		sb.WriteString(fmt.Sprintf("[%s]\n%s\n\n", role, content))
	}

	sb.WriteString("---\n\n")
	sb.WriteString("Now provide the summary.")

	return sb.String()
}

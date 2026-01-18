package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// confirmOrCommentToAI shows a y/n/c confirmation.
// If the user selects comment (c), it will add the comment to history and return handled=true.
// The comment will be processed in the next API call, avoiding recursive chat() calls.
// This implements the global UX rule: whenever the app asks for confirmation, the user can comment,
// and commenting should trigger an AI re-proposal (Plan A).
func confirmOrCommentToAI(agent *Agent, prompt string, commentContext string) (yes bool, handled bool) {
	res := tools.ConfirmInteractive(prompt)
	switch res.Action {
	case "yes":
		return true, false
	case "no":
		return false, false
	case "comment":
		if agent == nil {
			// No agent available (should be rare). Treat as cancelled.
			return false, false
		}

		// Build a clear instruction for the assistant.
		msg := "[USER COMMENT]\n"
		if commentContext != "" {
			msg += fmt.Sprintf("Context: %s\n", commentContext)
		}
		if res.Comment != "" {
			msg += fmt.Sprintf("Comment:\n%s\n", res.Comment)
		}
		msg += "\nPlease propose the next best action based on this feedback. Do not execute any tool yet; wait for confirmation."

		// コメントを履歴に追加（chat()を再帰呼び出ししない）
		// 次のツール実行ループで自然に処理される
		agent.History = append(agent.History, api.Message{
			Role:    "user",
			Content: msg,
		})

		yellow.Println("💬 Comment added. AI will consider it in the next response.")
		return false, true
	default:
		return false, false
	}
}

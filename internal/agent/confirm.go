package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// confirmOrCommentToAI shows a y/n/c confirmation.
// If the user selects comment (c), it will send the comment back to the AI and return handled=true.
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
		msg := "User left a comment instead of confirming.\n"
		if commentContext != "" {
			msg += fmt.Sprintf("Context: %s\n", commentContext)
		}
		if res.Comment != "" {
			msg += fmt.Sprintf("Comment:\n%s\n", res.Comment)
		}
		msg += "Please propose the next best action. Do not execute any tool yet; wait for confirmation."

		agent.chat(msg)
		return false, true
	default:
		return false, false
	}
}

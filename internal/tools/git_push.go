package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeGitPush は git push を実行
func executeGitPush() string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("🚀 Git Push / リモートへプッシュ\n")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("⚠️  Warning: Changes will be published to remote / 警告: リモートリポジトリに変更が公開されます")

	dec := confirmWithAutoApproveDecision("git_push", "Push to remote? / プッシュしますか？")
	switch dec.Action {
	case ConfirmYes:
		// continue
	case ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for git_push.

Comment:
%s

Next actions:
- Verify git status/diff before pushing.
- Adjust the plan (e.g., commit message, branch) and propose again.

IMPORTANT: Do NOT push until the user approves.`, strings.TrimSpace(dec.Comment))
	default:
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	if result == "" {
		result = "✅ Pushed successfully"
	}
	fmt.Println(result)
	return result
}

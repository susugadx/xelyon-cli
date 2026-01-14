package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeGitCommit は git commit を実行
func executeGitCommit(message string) string {
	if message == "" {
		return "Error: commit message is required"
	}

	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("💾 Git Commit / Gitコミット\n")
	cyan.Printf("📝 Message / メッセージ:\n%s\n", message)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	dec := Confirm("Commit with this message? / この内容でコミットしますか？")
	switch dec.Action {
	case ConfirmYes:
		// continue
	case ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for git_commit.

Comment:
%s

Next actions:
- Revise the commit message if needed.
- Or stage different files before committing.

IMPORTANT: Do NOT create a commit until the user approves.`, strings.TrimSpace(dec.Comment))
	default:
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	result := string(output)
	fmt.Println(result)
	return result
}

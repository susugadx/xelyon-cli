package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ExecuteGitCommit executes git commit
func ExecuteGitCommit(message string) string {
	if message == "" {
		return "Error: commit message is required"
	}

	// Confirmation UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("💾 Git Commit / Gitコミット\n")
	cyan.Printf("📝 Message / メッセージ:\n%s\n", message)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	dec := common.Confirm("Commit with this message? / この内容でコミットしますか？")
	switch dec.Action {
	case common.ConfirmYes:
		// continue
	case common.ConfirmComment:
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

package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeGitAdd は git add を実行
func executeGitAdd(path string) string {
	// 確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("➕ Git Stage / Gitステージング\n")
	cyan.Printf("📂 Path / パス: %s\n", path)
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	dec := Confirm("Stage this file? / このファイルをステージングしますか？")
	switch dec.Action {
	case ConfirmYes:
		// continue
	case ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for git_add.

Comment:
%s

Next actions:
- Verify the correct path(s) to stage.
- Consider staging a narrower set of files.

IMPORTANT: Do NOT stage files until the user approves.`, strings.TrimSpace(dec.Comment))
	default:
		return "Cancelled by user"
	}

	cmd := exec.Command("git", "add", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	return fmt.Sprintf("✅ Staged: %s", path)
}

package tools

import (
	"fmt"
	"strings"
)

// executeGitBranch はブランチ操作（list/create/switch）
func executeGitBranch(action, branchName string) string {
	// デフォルトはlist
	if action == "" {
		action = "list"
	}

	// Validation
	if (action == "create" || action == "switch") && branchName == "" {
		return fmt.Sprintf("Error: branch_name is required for action '%s'", action)
	}

	// Action: list
	if action == "list" {
		green.Println("📋 git branch")
		output, err := ExecuteGitCommand("branch", "-a")
		if err != nil {
			return FormatGitError(err, output)
		}
		if output == "" {
			return "No branches found"
		}
		return output
	}

	// Action: create
	if action == "create" {
		green.Printf("🌿 Creating branch: %s\n", branchName)
		output, err := ExecuteGitCommand("branch", branchName)
		if err != nil {
			return FormatGitError(err, output)
		}
		return FormatGitSuccess("Created branch", branchName)
	}

	// Action: switch
	if action == "switch" {
		// 未コミット変更チェック
		status, hasChanges, _ := CheckUncommittedChanges()

		if hasChanges {
			// 確認UI
			DisplayGitConfirmHeader(
				"🔀 Git Branch Switch / ブランチ切り替え",
				"🌿 Target Branch / 対象ブランチ",
				branchName,
			)
			DisplayUncommittedChangesWarning(status)

			dec := Confirm("Switch branch anyway? / それでもブランチを切り替えますか？")
			switch dec.Action {
			case ConfirmYes:
				// continue
			case ConfirmComment:
				return fmt.Sprintf(`[COMMENT] User provided feedback for git_branch switch.

Comment:
%s

Next actions:
- Consider stashing or committing changes before switching.
- Or switch to a different branch.

IMPORTANT: Do NOT switch branches until the user approves.`, strings.TrimSpace(dec.Comment))
			default:
				return "Cancelled by user"
			}
		} else {
			green.Printf("🔀 Switching to branch: %s\n", branchName)
		}

		output, err := ExecuteGitCommand("checkout", branchName)
		if err != nil {
			return FormatGitError(err, output)
		}
		return fmt.Sprintf("✅ Switched to branch: %s\n%s", branchName, output)
	}

	return fmt.Sprintf("Error: Unknown action '%s' (use 'list', 'create', or 'switch')", action)
}

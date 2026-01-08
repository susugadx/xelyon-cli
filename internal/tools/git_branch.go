package tools

import (
	"fmt"
	"os/exec"
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
		cmd := exec.Command("git", "branch", "-a")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		result := string(output)
		if result == "" {
			result = "No branches found"
		}
		return result
	}

	// Action: create
	if action == "create" {
		green.Printf("🌿 Creating branch: %s\n", branchName)
		cmd := exec.Command("git", "branch", branchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		return fmt.Sprintf("✅ Created branch: %s", branchName)
	}

	// Action: switch
	if action == "switch" {
		// 未コミット変更チェック
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusOutput, _ := statusCmd.CombinedOutput()
		hasChanges := len(strings.TrimSpace(string(statusOutput))) > 0

		if hasChanges {
			// 確認UI
			cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			cyan.Printf("🔀 Git Branch Switch / ブランチ切り替え\n")
			cyan.Printf("🌿 Target Branch / 対象ブランチ: %s\n", branchName)
			cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			yellow.Println("⚠️  Warning: You have uncommitted changes / 警告: コミットされていない変更があります")
			fmt.Println("\nUncommitted changes / 未コミットの変更:")
			fmt.Println(string(statusOutput))

			if !confirm("Switch branch anyway? / それでもブランチを切り替えますか？") {
				return "Cancelled by user"
			}
		} else {
			green.Printf("🔀 Switching to branch: %s\n", branchName)
		}

		cmd := exec.Command("git", "checkout", branchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		return fmt.Sprintf("✅ Switched to branch: %s\n%s", branchName, string(output))
	}

	return fmt.Sprintf("Error: Unknown action '%s' (use 'list', 'create', or 'switch')", action)
}

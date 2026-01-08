package tools

import (
	"fmt"
	"os/exec"
	"strings"
)

// executeGitStash はスタッシュ操作（save/list/pop/apply/drop）
func executeGitStash(action, message string) string {
	// デフォルトはsave
	if action == "" {
		action = "save"
	}

	// Validation
	validActions := map[string]bool{
		"save": true, "list": true, "pop": true, "apply": true, "drop": true,
	}
	if !validActions[action] {
		return fmt.Sprintf("Error: Unknown action '%s' (use 'save', 'list', 'pop', 'apply', or 'drop')", action)
	}

	// Action: save
	if action == "save" {
		// 変更があるかチェック
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusOutput, _ := statusCmd.CombinedOutput()
		if len(strings.TrimSpace(string(statusOutput))) == 0 {
			yellow.Println("⚠️  No changes to stash / スタッシュする変更がありません")
			return "No changes to stash (working tree clean)"
		}

		green.Println("📦 Stashing changes / 変更をスタッシュ")
		yellow.Println("\nChanges to stash / スタッシュする変更:")
		fmt.Println(string(statusOutput))

		args := []string{"stash", "push"}
		if message != "" {
			args = append(args, "-m", message)
		}

		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Stashed changes\n%s", string(output))
	}

	// Action: list
	if action == "list" {
		green.Println("📋 git stash list")
		cmd := exec.Command("git", "stash", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}
		result := string(output)
		if result == "" {
			result = "No stashes found"
		}
		return result
	}

	// Action: pop
	if action == "pop" {
		stashRef := "stash@{0}" // デフォルト: 最新
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
		showOutput, showErr := showCmd.CombinedOutput()

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📦 Git Stash Pop / スタッシュ適用・削除\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			lines := strings.Split(string(showOutput), "\n")
			maxLines := 20
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			if len(lines) > 20 {
				yellow.Printf("... (%d more lines)\n", len(lines)-20)
			}
		}

		if !confirm("Pop this stash? / このスタッシュを適用・削除しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "pop", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Popped stash: %s\n%s", stashRef, string(output))
	}

	// Action: apply
	if action == "apply" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
		showOutput, showErr := showCmd.CombinedOutput()

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("📦 Git Stash Apply / スタッシュ適用\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")
		yellow.Println("ℹ️  Note: Stash will be kept after apply / 注意: スタッシュは適用後も保持されます")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			lines := strings.Split(string(showOutput), "\n")
			maxLines := 20
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			if len(lines) > 20 {
				yellow.Printf("... (%d more lines)\n", len(lines)-20)
			}
		}

		if !confirm("Apply this stash? / このスタッシュを適用しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "apply", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Applied stash: %s\n%s", stashRef, string(output))
	}

	// Action: drop
	if action == "drop" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// 確認UI
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Printf("🗑️  Git Stash Drop / スタッシュ削除\n")
		cyan.Printf("📋 Stash / スタッシュ: %s\n", stashRef)
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		red.Println("⚠️  DESTRUCTIVE: This stash will be permanently deleted!")
		red.Println("⚠️  破壊的操作: このスタッシュは完全に削除されます!")

		if !confirm("Delete this stash? / このスタッシュを削除しますか？") {
			return "Cancelled by user"
		}

		cmd := exec.Command("git", "stash", "drop", stashRef)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %v\n%s", err, string(output))
		}

		return fmt.Sprintf("✅ Dropped stash: %s\n%s", stashRef, string(output))
	}

	return fmt.Sprintf("Error: Unknown action '%s'", action)
}

package tools

import (
	"fmt"
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
		status, hasChanges, _ := CheckUncommittedChanges()
		if !hasChanges {
			yellow.Println("⚠️  No changes to stash / スタッシュする変更がありません")
			return "No changes to stash (working tree clean)"
		}

		green.Println("📦 Stashing changes / 変更をスタッシュ")
		yellow.Println("\nChanges to stash / スタッシュする変更:")
		fmt.Println(status)

		args := []string{"stash", "push"}
		if message != "" {
			args = append(args, "-m", message)
		}

		output, err := ExecuteGitCommand(args...)
		if err != nil {
			return FormatGitError(err, output)
		}

		return fmt.Sprintf("✅ Stashed changes\n%s", output)
	}

	// Action: list
	if action == "list" {
		green.Println("📋 git stash list")
		output, err := ExecuteGitCommand("stash", "list")
		if err != nil {
			return FormatGitError(err, output)
		}
		if output == "" {
			return "No stashes found"
		}
		return output
	}

	// Action: pop
	if action == "pop" {
		stashRef := "stash@{0}" // デフォルト: 最新
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showOutput, showErr := ExecuteGitCommand("stash", "show", "-p", stashRef)

		// 確認UI
		DisplayGitConfirmHeader(
			"📦 Git Stash Pop / スタッシュ適用・削除",
			"📋 Stash / スタッシュ",
			stashRef,
		)
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			fmt.Println(TruncateOutput(showOutput, 20))
		}

		dec := Confirm("Pop this stash? / このスタッシュを適用・削除しますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for git_stash pop.

Comment:
%s

Next actions:
- Verify which stash index to use.
- Consider using apply instead of pop if you want to keep the stash.

IMPORTANT: Do NOT apply/pop the stash until the user approves.`, strings.TrimSpace(dec.Comment))
		default:
			return "Cancelled by user"
		}

		output, err := ExecuteGitCommand("stash", "pop", stashRef)
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, output)
		}

		return fmt.Sprintf("✅ Popped stash: %s\n%s", stashRef, output)
	}

	// Action: apply
	if action == "apply" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// スタッシュプレビュー取得
		showOutput, showErr := ExecuteGitCommand("stash", "show", "-p", stashRef)

		// 確認UI
		DisplayGitConfirmHeader(
			"📦 Git Stash Apply / スタッシュ適用",
			"📋 Stash / スタッシュ",
			stashRef,
		)
		yellow.Println("⚠️  Warning: This may cause merge conflicts / 警告: マージ競合が発生する可能性があります")
		yellow.Println("ℹ️  Note: Stash will be kept after apply / 注意: スタッシュは適用後も保持されます")

		if showErr == nil {
			yellow.Println("\nStash preview / スタッシュプレビュー (first 20 lines):")
			fmt.Println(TruncateOutput(showOutput, 20))
		}

		dec := Confirm("Apply this stash? / このスタッシュを適用しますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for git_stash apply.

Comment:
%s

Next actions:
- Verify which stash index to apply.
- Review the stash diff preview before applying.

IMPORTANT: Do NOT apply the stash until the user approves.`, strings.TrimSpace(dec.Comment))
		default:
			return "Cancelled by user"
		}

		output, err := ExecuteGitCommand("stash", "apply", stashRef)
		if err != nil {
			return fmt.Sprintf("Error (may have conflicts): %v\n%s", err, output)
		}

		return fmt.Sprintf("✅ Applied stash: %s\n%s", stashRef, output)
	}

	// Action: drop
	if action == "drop" {
		stashRef := "stash@{0}"
		if message != "" {
			stashRef = "stash@{" + message + "}"
		}

		// 確認UI
		DisplayGitConfirmHeader(
			"🗑️  Git Stash Drop / スタッシュ削除",
			"📋 Stash / スタッシュ",
			stashRef,
		)
		red.Println("⚠️  DESTRUCTIVE: This stash will be permanently deleted!")
		red.Println("⚠️  破壊的操作: このスタッシュは完全に削除されます!")

		dec := Confirm("Delete this stash? / このスタッシュを削除しますか？")
		switch dec.Action {
		case ConfirmYes:
			// continue
		case ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for git_stash drop.

Comment:
%s

Next actions:
- Double-check the stash index.
- Consider keeping the stash and using apply/pop instead.

IMPORTANT: Do NOT delete the stash until the user approves.`, strings.TrimSpace(dec.Comment))
		default:
			return "Cancelled by user"
		}

		output, err := ExecuteGitCommand("stash", "drop", stashRef)
		if err != nil {
			return FormatGitError(err, output)
		}

		return fmt.Sprintf("✅ Dropped stash: %s\n%s", stashRef, output)
	}

	return fmt.Sprintf("Error: Unknown action '%s'", action)
}

package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// executeGitAdd は git add を実行
// pathはスペース区切りで複数ファイルを指定可能
// 複数ファイルの場合、バッチ確認UIを表示
func executeGitAdd(path string) string {
	if path == "" {
		path = "."
	}

	// パスを分割（スペース区切り、ただしクォート内はそのまま）
	paths := splitPaths(path)

	// 単一ファイル（または単一パターン）の場合は従来の確認UI
	if len(paths) == 1 {
		return executeGitAddSingle(paths[0])
	}

	// 複数ファイルの場合はバッチ確認UI
	return executeGitAddBatch(paths)
}

// splitPaths はスペース区切りのパスを分割（シンプル実装）
func splitPaths(path string) []string {
	// カンマ区切りも対応
	path = strings.ReplaceAll(path, ",", " ")
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return []string{"."}
	}
	return parts
}

// executeGitAddSingle は単一ファイルの git add を実行（従来の動作）
func executeGitAddSingle(path string) string {
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

// executeGitAddBatch は複数ファイルのバッチ確認UIを表示
func executeGitAddBatch(paths []string) string {
	cfg := config.GetGlobalConfig()

	// バッチ確認が無効の場合は従来の1ファイルずつ確認
	if !cfg.GitStage.BatchConfirm {
		var results []string
		for _, p := range paths {
			result := executeGitAddSingle(p)
			results = append(results, result)
			// キャンセルされたら中断
			if strings.Contains(result, "Cancelled") || strings.Contains(result, "[COMMENT]") {
				return strings.Join(results, "\n")
			}
		}
		return strings.Join(results, "\n")
	}

	// バッチ確認UI
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Printf("📦 Git Stage Batch / Gitバッチステージング\n")
	cyan.Printf("   %d files / %dファイル\n", len(paths), len(paths))
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, p := range paths {
		fmt.Printf("  - %s\n", p)
	}

	fmt.Println()
	yellow.Print("Stage all? (y/n/s=select) / 全てステージング？ (y/n/s=個別選択): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "Cancelled by user (read error)"
	}
	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
	case "y", "yes", "はい":
		// 全ファイルをステージング
		return stageAllFiles(paths)

	case "s", "select", "選択":
		// 個別選択モード
		return stageFilesSelectively(paths)

	case "n", "no", "いいえ":
		return "Cancelled by user"

	default:
		// 無効な入力はキャンセル扱い
		yellow.Println("Invalid input. Cancelled.")
		return "Cancelled by user"
	}
}

// stageAllFiles は全ファイルを一括ステージング
func stageAllFiles(paths []string) string {
	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}

	green.Printf("✅ Staged %d files:\n", len(paths))
	for _, p := range paths {
		fmt.Printf("  - %s\n", p)
	}
	return fmt.Sprintf("✅ Staged %d files", len(paths))
}

// stageFilesSelectively は個別選択モードでファイルをステージング
func stageFilesSelectively(paths []string) string {
	var staged []string
	var skipped []string

	for _, p := range paths {
		yellow.Printf("Stage '%s'? (y/n): ", p)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			skipped = append(skipped, p)
			continue
		}
		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			cmd := exec.Command("git", "add", p)
			output, err := cmd.CombinedOutput()
			if err != nil {
				red.Printf("Error staging %s: %v\n%s\n", p, err, string(output))
				skipped = append(skipped, p)
			} else {
				green.Printf("✅ Staged: %s\n", p)
				staged = append(staged, p)
			}
		} else {
			skipped = append(skipped, p)
		}
	}

	// 結果サマリー
	var result strings.Builder
	if len(staged) > 0 {
		result.WriteString(fmt.Sprintf("✅ Staged %d files: %s\n", len(staged), strings.Join(staged, ", ")))
	}
	if len(skipped) > 0 {
		result.WriteString(fmt.Sprintf("⏭️ Skipped %d files: %s", len(skipped), strings.Join(skipped, ", ")))
	}
	if len(staged) == 0 && len(skipped) == 0 {
		return "No files to stage"
	}
	return strings.TrimSpace(result.String())
}

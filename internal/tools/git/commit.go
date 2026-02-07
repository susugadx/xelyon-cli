package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/lsp"
)

// ExecuteGitCommit executes git commit.
// skipDiagnostics が true の場合、コミット前の LSP 診断チェックをスキップする。
func ExecuteGitCommit(message string, skipDiagnostics bool) string {
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

	// コミット前の LSP 診断チェック
	if !skipDiagnostics {
		if result := runPreCommitDiagnostics(); result != "" {
			return result
		}
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

// runPreCommitDiagnostics はステージされた変更ファイルに対して LSP 診断を実行する。
// エラーがあればブロックメッセージを返し、問題なければ空文字を返す。
func runPreCommitDiagnostics() string {
	// ステージされたファイル一覧を取得
	files, err := getStagedFiles()
	if err != nil || len(files) == 0 {
		return ""
	}

	cyan.Printf("🔍 Running LSP diagnostics on %d staged file(s)...\n", len(files))

	result := lsp.CheckDiagnosticsForFiles(files)

	// 問題なし
	if result.ErrorCount == 0 && result.WarnCount == 0 {
		common.Green.Println("✅ No LSP errors found")
		return ""
	}

	// Warning のみ → 表示するがブロックしない
	if !result.HasErrors {
		yellow.Printf("⚠️ %d warning(s) found (commit not blocked)\n", result.WarnCount)
		fmt.Println(result.Summary)
		return ""
	}

	// Error あり → ブロック
	common.Red.Printf("❌ Commit blocked: %d error(s) found\n", result.ErrorCount)
	fmt.Println(result.Summary)

	return fmt.Sprintf(`[BLOCKED] Commit blocked by LSP diagnostics.

%s
Fix the errors above before committing.
Use --no-diagnostics to skip this check if the errors are false positives.`, result.Summary)
}

// getStagedFiles は git diff --cached --name-only でステージされたファイル一覧を取得する。
func getStagedFiles() ([]string, error) {
	output, err := ExecuteGitCommand("diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

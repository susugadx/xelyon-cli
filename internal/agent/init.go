package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleInitCommand は /init コマンドを処理（AGENTS.md 生成）。
func handleInitCommand(agent *Agent) bool {
	runtimeUI := agent.ui()
	out := runtimeUI.Output()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📝 AGENTS.md Guidance Generator")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	agentsPath, err := config.CreateDefaultProjectAgentInstructionsTemplate()
	if errors.Is(err, config.ErrProjectAgentInstructionsExists) {
		displayPath := displayInitPath(agentsPath)
		yellow.Fprintln(out, "⚠️  AGENTS.md already exists. Left unchanged.")
		yellow.Fprintf(out, "   Path: %s\n", displayPath)
		yellow.Fprintln(out, "   Edit it directly, or use /config to choose additional guidance files.")
		return true
	}
	if err != nil {
		red.Fprintf(out, "Failed to write AGENTS.md: %v\n", err)
		return true
	}

	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Fprintln(out, "✅ AGENTS.md created!")
	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)
	displayPath := displayInitPath(agentsPath)
	yellow.Fprintf(out, "Path: %s\n", displayPath)
	if initRootFileExists(agentsPath, "CLAUDE.md") || initRootFileExists(agentsPath, ".claude/CLAUDE.md") {
		yellow.Fprintln(out, "Existing CLAUDE guidance was left unchanged. Use /config to include it when needed.")
	}
	yellow.Fprintln(out, "Next steps:")
	yellow.Fprintf(out, "  1. Edit %s with repo guidance for agents\n", displayPath)
	yellow.Fprintf(out, "  2. If this repository uses git, run: git add %s\n", displayPath)
	yellow.Fprintln(out, "  3. Use /config to choose project/global guidance files")
	yellow.Fprintln(out, "  4. Use /project when you need XELYON-specific repo config such as ignore or final_checks")

	return true
}

// fileExists はファイルの存在を確認（他のコマンドでも使用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func initRootFileExists(agentsPath, relativePath string) bool {
	if agentsPath == "" {
		return fileExists(relativePath)
	}
	return fileExists(filepath.Join(filepath.Dir(agentsPath), filepath.FromSlash(relativePath)))
}

func displayInitPath(path string) string {
	if path == "" {
		return "AGENTS.md"
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || rel == "" {
		return path
	}
	return filepath.ToSlash(rel)
}

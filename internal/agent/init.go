package agent

import (
	"errors"
	"fmt"
	"os"

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

	err := config.CreateProjectAgentInstructionsTemplate("")
	if errors.Is(err, config.ErrProjectAgentInstructionsExists) {
		yellow.Fprintln(out, "⚠️  AGENTS.md already exists. Left unchanged.")
		yellow.Fprintln(out, "   Edit AGENTS.md directly, or use /config to choose additional guidance files.")
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
	if fileExists("CLAUDE.md") || fileExists(".claude/CLAUDE.md") {
		yellow.Fprintln(out, "Existing CLAUDE guidance was left unchanged. Use /config to include it when needed.")
	}
	yellow.Fprintln(out, "Next steps:")
	yellow.Fprintln(out, "  1. Edit AGENTS.md with repo guidance for agents")
	yellow.Fprintln(out, "  2. Use /config to choose project/global guidance files")
	yellow.Fprintln(out, "  3. Use /project when you need XELYON-specific repo config such as ignore or final_checks")

	return true
}

// fileExists はファイルの存在を確認（他のコマンドでも使用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

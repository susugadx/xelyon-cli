package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type initCommandOptions struct {
	allowOverwritePrompt bool
}

// handleInitCommand は /init コマンドを処理（xelyon.yaml 生成）。
func handleInitCommand(agent *Agent) bool {
	return handleInitCommandWithOptions(agent, initCommandOptions{allowOverwritePrompt: true})
}

func handleInitCommandWithOptions(agent *Agent, opts initCommandOptions) bool {
	runtimeUI := agent.ui()
	promptIO := runtimeUI.PromptIO()
	out := runtimeUI.Output()

	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Fprintln(out, "📝 xelyon.yaml Template Generator")
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)

	err := config.CreateProjectConfigTemplate("", false)
	if errors.Is(err, config.ErrProjectConfigExists) {
		yellow.Fprintln(out, "⚠️  xelyon.yaml already exists")
		if !opts.allowOverwritePrompt {
			yellow.Fprintln(out, "   Not overwriting from TUI mode. Edit or remove xelyon.yaml and run /init again.")
			return true
		}
		_, _ = fmt.Fprint(out, "Overwrite? (y/n): ")
		var input string
		input, err = promptIO.ReadSimpleLine()
		if err != nil {
			red.Fprintf(out, "Failed to read input: %v\n", err)
			return true
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			yellow.Fprintln(out, "Cancelled")
			return true
		}
		err = config.CreateProjectConfigTemplate("", true)
	}
	if err != nil {
		red.Fprintf(out, "Failed to write xelyon.yaml: %v\n", err)
		return true
	}

	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Fprintln(out, "✅ xelyon.yaml template created!")
	green.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(out)
	yellow.Fprintln(out, "Next steps:")
	yellow.Fprintln(out, "  - xelyon.yaml is optional.")
	yellow.Fprintln(out, "  - XELYON can also use existing AGENTS.md / CLAUDE.md guidance.")
	yellow.Fprintln(out, "  - Create xelyon.yaml when you want XELYON-specific runtime settings such as conditional rules, ignore patterns, or final_checks.")
	yellow.Fprintln(out, "  1. Edit xelyon.yaml to add your project context and rules")
	yellow.Fprintln(out, "  2. Optionally add conditional rules or shared ignore patterns")
	yellow.Fprintln(out, "  3. Optionally configure final_checks.commands for project checks")
	yellow.Fprintln(out, "  4. xelyon.yaml will be automatically loaded on next session")

	return true
}

// fileExists はファイルの存在を確認（他のコマンドでも使用）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

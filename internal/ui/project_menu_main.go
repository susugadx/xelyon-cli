package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectMainMenuSnapshot struct {
	contextPreview     string
	rulesCount         int
	finalChecksPreview string
}

func buildProjectMainMenuSnapshot(pc *config.ProjectConfig) projectMainMenuSnapshot {
	contextPreview := "(empty)"
	if pc != nil {
		trimmed := strings.TrimSpace(pc.Context)
		if trimmed != "" {
			contextPreview = truncateString(trimmed, 30)
		}
	}

	rulesCount := 0
	if pc != nil {
		rulesCount = len(pc.Rules)
	}

	finalChecksPreview := "(not set)"
	if pc != nil && pc.FinalChecks != nil && len(pc.FinalChecks.Commands) > 0 {
		finalChecksPreview = fmt.Sprintf("(%d cmds)", len(pc.FinalChecks.Commands))
	}

	return projectMainMenuSnapshot{
		contextPreview:     contextPreview,
		rulesCount:         rulesCount,
		finalChecksPreview: finalChecksPreview,
	}
}

// Run はメインメニューを表示し、編集結果を返す。
// changed=true の場合、呼び出し元で SaveProjectConfig() を実行すること。
func (m *ProjectMenu) Run() (changed bool, err error) {
	ctx := newProjectPromptContext(m.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	for {
		snapshot := buildProjectMainMenuSnapshot(m.PC)
		m.renderProjectMainMenu(out, snapshot)

		action := resolveProjectMainMenuAction(readProjectMenuInput(&promptIO))
		switch action {
		case projectMainMenuShowContext:
			m.showContext()
		case projectMainMenuEditRules:
			m.editRules()
		case projectMainMenuEditFinalChecks:
			m.editFinalChecks()
		case projectMainMenuSave:
			return m.changed, nil
		case projectMainMenuCancel:
			return false, nil
		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use 1-3/s/c%s\n", colorDim, colorReset)
		}
	}
}

func (m *ProjectMenu) renderProjectMainMenu(out io.Writer, snapshot projectMainMenuSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── Project Settings (xelyon.yaml) ───────────%s\n\n", colorCyan, colorReset)
	_, _ = fmt.Fprintf(out, "  [1] 📝 Context      %s\n", snapshot.contextPreview)
	_, _ = fmt.Fprintf(out, "  [2] 📋 Rules        (%d items)\n", snapshot.rulesCount)
	_, _ = fmt.Fprintf(out, "  [3] 🧪 Final Checks %s\n", snapshot.finalChecksPreview)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [s] Save and exit")
	_, _ = fmt.Fprintln(out, "  [c] Cancel")
	_, _ = fmt.Fprintf(out, "\n%sSelect:%s ", colorCyan, colorReset)
}

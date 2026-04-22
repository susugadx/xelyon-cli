package ui

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectFinalChecksSnapshot struct {
	commandsInfo string
	timeout      int
}

func buildProjectFinalChecksSnapshot(cfg *config.FinalChecksConfig) projectFinalChecksSnapshot {
	commandsInfo := "(empty)"
	if cfg != nil && len(cfg.Commands) > 0 {
		commandsInfo = fmt.Sprintf("(%d cmds)", len(cfg.Commands))
	}

	timeout := 600
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	return projectFinalChecksSnapshot{
		commandsInfo: commandsInfo,
		timeout:      timeout,
	}
}

func (m *ProjectMenu) ensureFinalChecksConfig() {
	if m.PC.FinalChecks == nil {
		m.PC.FinalChecks = &config.FinalChecksConfig{}
	}
}

func (m *ProjectMenu) clearEmptyFinalChecksConfig() {
	if m.PC.FinalChecks == nil {
		return
	}
	if len(m.PC.FinalChecks.Commands) == 0 && m.PC.FinalChecks.Timeout == 0 {
		m.PC.FinalChecks = nil
	}
}

// editFinalChecks は Final Checks のサブメニューを表示する。
func (m *ProjectMenu) editFinalChecks() {
	m.ensureFinalChecksConfig()

	promptIO := runtimeOrDefault(m.Runtime).PromptIO()
	out := promptIO.Out

	for {
		snapshot := buildProjectFinalChecksSnapshot(m.PC.FinalChecks)
		m.renderProjectFinalChecksMenu(out, snapshot)

		action := resolveProjectFinalChecksAction(readProjectMenuInput(&promptIO))
		switch action {
		case projectFinalChecksEditCommands:
			m.editFinalChecksCommands()
		case projectFinalChecksEditTimeout:
			m.editFinalChecksTimeout(&promptIO, out, snapshot.timeout)
		case projectFinalChecksBack:
			m.clearEmptyFinalChecksConfig()
			return
		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use 1-2/b%s\n", colorDim, colorReset)
		}
	}
}

func (m *ProjectMenu) renderProjectFinalChecksMenu(out io.Writer, snapshot projectFinalChecksSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── Final Checks ────────────────────────────%s\n\n", colorCyan, colorReset)
	_, _ = fmt.Fprintf(out, "  [1] commands        %s\n", snapshot.commandsInfo)
	_, _ = fmt.Fprintf(out, "  [2] timeout         %d sec\n", snapshot.timeout)
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [b] Back")
	_, _ = fmt.Fprintf(out, "\n%sSelect:%s ", colorCyan, colorReset)
}

func (m *ProjectMenu) editFinalChecksCommands() {
	editor := NewStringSliceEditorWithRuntime("final_checks.commands", m.PC.FinalChecks.Commands, m.Runtime)
	result, saved, _ := editor.Run()
	if !saved {
		return
	}
	m.PC.FinalChecks.Commands = result
	m.changed = true
}

func (m *ProjectMenu) editFinalChecksTimeout(promptIO *PromptIO, out io.Writer, currentTimeout int) {
	_, _ = fmt.Fprintf(out, "Enter timeout (seconds, current: %d): ", currentTimeout)
	input := readLineWithIO(promptIO)
	timeout, ok := parsePositiveIntInput(input)
	if !ok {
		return
	}
	m.PC.FinalChecks.Timeout = timeout
	m.changed = true
	_, _ = fmt.Fprintf(out, "%s✓ timeout = %d%s\n", colorGreen, timeout, colorReset)
}

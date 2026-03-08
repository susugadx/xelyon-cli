package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// ProjectMenu は xelyon.yaml の対話式編集メニュー
type ProjectMenu struct {
	PC      *config.ProjectConfig
	changed bool
	Runtime *Runtime
}

// NewProjectMenu は新しい ProjectMenu を作成
func NewProjectMenu(pc *config.ProjectConfig) *ProjectMenu {
	return NewProjectMenuWithRuntime(pc, DefaultRuntime())
}

// NewProjectMenuWithRuntime は UI runtime を指定して新しい ProjectMenu を作成する。
func NewProjectMenuWithRuntime(pc *config.ProjectConfig, runtime *Runtime) *ProjectMenu {
	return &ProjectMenu{PC: pc, Runtime: runtimeOrDefault(runtime)}
}

// Run はメインメニューを表示し、編集結果を返す。
// changed=true の場合、呼び出し元で SaveProjectConfig() を実行すること。
func (m *ProjectMenu) Run() (changed bool, err error) {
	runtime := runtimeOrDefault(m.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	for {
		// メインメニュー表示
		_, _ = fmt.Fprintf(out, "\n%s── Project Settings (xelyon.yaml) ───────────%s\n\n", colorCyan, colorReset)

		// 1. Context (readonly)
		ctxPreview := truncateString(strings.TrimSpace(m.PC.Context), 30)
		if ctxPreview == "" {
			ctxPreview = "(empty)"
		}
		_, _ = fmt.Fprintf(out, "  [1] 📝 Context      %s\n", ctxPreview)

		// 2. Rules
		_, _ = fmt.Fprintf(out, "  [2] 📋 Rules        (%d items)\n", len(m.PC.Rules))

		// 3. Hooks
		hooksPreview := "(not set)"
		if m.PC.Hooks != nil && len(m.PC.Hooks.OnCompletion) > 0 {
			hooksPreview = fmt.Sprintf("(%d cmds)", len(m.PC.Hooks.OnCompletion))
		}
		_, _ = fmt.Fprintf(out, "  [3] 🔧 Hooks        %s\n", hooksPreview)

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [s] Save and exit")
		_, _ = fmt.Fprintln(out, "  [c] Cancel")
		_, _ = fmt.Fprintf(out, "\n%sSelect:%s ", colorCyan, colorReset)

		input := strings.TrimSpace(strings.ToLower(readLineWithIO(&promptIO)))

		switch input {
		case "1":
			m.showContext()
		case "2":
			m.editRules()
		case "3":
			m.editHooks()
		case "s", "save":
			return m.changed, nil
		case "c", "cancel":
			return false, nil
		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use 1-3/s/c%s\n", colorDim, colorReset)
		}
	}
}

// showContext は Context を表示する（編集不可）
func (m *ProjectMenu) showContext() {
	out := runtimeOrDefault(m.Runtime).Output()
	_, _ = fmt.Fprintf(out, "\n%s── Context ──────────────────────────────────%s\n", colorCyan, colorReset)
	if m.PC.Context == "" {
		_, _ = fmt.Fprintf(out, "  %s(empty)%s\n", colorDim, colorReset)
	} else {
		for _, line := range strings.Split(strings.TrimRight(m.PC.Context, "\n"), "\n") {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}
	}
	_, _ = fmt.Fprintf(out, "%s─────────────────────────────────────────────%s\n", colorCyan, colorReset)
	_, _ = fmt.Fprintf(out, "  %sEdit xelyon.yaml directly to change context%s\n", colorDim, colorReset)
}

// editRules は Rules を StringSliceEditor で編集する
func (m *ProjectMenu) editRules() {
	editor := NewStringSliceEditorWithRuntime("rules", m.PC.Rules, m.Runtime)
	result, saved, _ := editor.Run()
	if saved {
		m.PC.Rules = result
		m.changed = true
	}
}

// editHooks は Hooks のサブメニューを表示する
func (m *ProjectMenu) editHooks() {
	if m.PC.Hooks == nil {
		m.PC.Hooks = &config.HooksConfig{}
	}

	promptIO := runtimeOrDefault(m.Runtime).PromptIO()
	out := promptIO.Out

	for {
		_, _ = fmt.Fprintf(out, "\n%s── Hooks ────────────────────────────────────%s\n\n", colorCyan, colorReset)

		// on_completion
		cmdInfo := "(empty)"
		if len(m.PC.Hooks.OnCompletion) > 0 {
			cmdInfo = fmt.Sprintf("(%d cmds)", len(m.PC.Hooks.OnCompletion))
		}
		_, _ = fmt.Fprintf(out, "  [1] on_completion   %s\n", cmdInfo)

		// timeout
		timeout := m.PC.Hooks.Timeout
		if timeout == 0 {
			timeout = 60
		}
		_, _ = fmt.Fprintf(out, "  [2] timeout         %d sec\n", timeout)

		// max_retry
		maxRetry := m.PC.Hooks.MaxRetry
		if maxRetry == 0 {
			maxRetry = 3
		}
		_, _ = fmt.Fprintf(out, "  [3] max_retry       %d\n", maxRetry)

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [b] Back")
		_, _ = fmt.Fprintf(out, "\n%sSelect:%s ", colorCyan, colorReset)

		input := strings.TrimSpace(strings.ToLower(readLineWithIO(&promptIO)))

		switch input {
		case "1":
			editor := NewStringSliceEditorWithRuntime("hooks.on_completion", m.PC.Hooks.OnCompletion, m.Runtime)
			result, saved, _ := editor.Run()
			if saved {
				m.PC.Hooks.OnCompletion = result
				m.changed = true
			}
		case "2":
			_, _ = fmt.Fprintf(out, "Enter timeout (seconds, current: %d): ", timeout)
			numStr := strings.TrimSpace(readLineWithIO(&promptIO))
			if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
				m.PC.Hooks.Timeout = num
				m.changed = true
				_, _ = fmt.Fprintf(out, "%s✓ timeout = %d%s\n", colorGreen, num, colorReset)
			}
		case "3":
			_, _ = fmt.Fprintf(out, "Enter max_retry (current: %d): ", maxRetry)
			numStr := strings.TrimSpace(readLineWithIO(&promptIO))
			if num, err := strconv.Atoi(numStr); err == nil && num >= 0 {
				m.PC.Hooks.MaxRetry = num
				m.changed = true
				_, _ = fmt.Fprintf(out, "%s✓ max_retry = %d%s\n", colorGreen, num, colorReset)
			}
		case "b", "back":
			// hooks が全て空なら nil に戻す
			if len(m.PC.Hooks.OnCompletion) == 0 && m.PC.Hooks.Timeout == 0 && m.PC.Hooks.MaxRetry == 0 {
				m.PC.Hooks = nil
			}
			return
		default:
			_, _ = fmt.Fprintf(out, "%sUnknown command. Use 1-3/b%s\n", colorDim, colorReset)
		}
	}
}

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
}

// NewProjectMenu は新しい ProjectMenu を作成
func NewProjectMenu(pc *config.ProjectConfig) *ProjectMenu {
	return &ProjectMenu{PC: pc}
}

// Run はメインメニューを表示し、編集結果を返す。
// changed=true の場合、呼び出し元で SaveProjectConfig() を実行すること。
func (m *ProjectMenu) Run() (changed bool, err error) {
	StopGlobalSpinner()
	fmt.Print("\033[?25h") // カーソル表示

	for {
		// メインメニュー表示
		fmt.Printf("\n%s── Project Settings (xelyon.yaml) ───────────%s\n\n", colorCyan, colorReset)

		// 1. Context (readonly)
		ctxPreview := truncateString(strings.TrimSpace(m.PC.Context), 30)
		if ctxPreview == "" {
			ctxPreview = "(empty)"
		}
		fmt.Printf("  [1] 📝 Context      %s\n", ctxPreview)

		// 2. Rules
		fmt.Printf("  [2] 📋 Rules        (%d items)\n", len(m.PC.Rules))

		// 3. Hooks
		hooksPreview := "(not set)"
		if m.PC.Hooks != nil && len(m.PC.Hooks.OnCompletion) > 0 {
			hooksPreview = fmt.Sprintf("(%d cmds)", len(m.PC.Hooks.OnCompletion))
		}
		fmt.Printf("  [3] 🔧 Hooks        %s\n", hooksPreview)

		fmt.Println()
		fmt.Println("  [s] Save and exit")
		fmt.Println("  [c] Cancel")
		fmt.Printf("\n%sSelect:%s ", colorCyan, colorReset)

		input := strings.TrimSpace(strings.ToLower(readLine()))

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
			fmt.Printf("%sUnknown command. Use 1-3/s/c%s\n", colorDim, colorReset)
		}
	}
}

// showContext は Context を表示する（編集不可）
func (m *ProjectMenu) showContext() {
	fmt.Printf("\n%s── Context ──────────────────────────────────%s\n", colorCyan, colorReset)
	if m.PC.Context == "" {
		fmt.Printf("  %s(empty)%s\n", colorDim, colorReset)
	} else {
		for _, line := range strings.Split(strings.TrimRight(m.PC.Context, "\n"), "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Printf("%s─────────────────────────────────────────────%s\n", colorCyan, colorReset)
	fmt.Printf("  %sEdit xelyon.yaml directly to change context%s\n", colorDim, colorReset)
}

// editRules は Rules を StringSliceEditor で編集する
func (m *ProjectMenu) editRules() {
	editor := NewStringSliceEditor("rules", m.PC.Rules)
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

	for {
		fmt.Printf("\n%s── Hooks ────────────────────────────────────%s\n\n", colorCyan, colorReset)

		// on_completion
		cmdInfo := "(empty)"
		if len(m.PC.Hooks.OnCompletion) > 0 {
			cmdInfo = fmt.Sprintf("(%d cmds)", len(m.PC.Hooks.OnCompletion))
		}
		fmt.Printf("  [1] on_completion   %s\n", cmdInfo)

		// timeout
		timeout := m.PC.Hooks.Timeout
		if timeout == 0 {
			timeout = 60
		}
		fmt.Printf("  [2] timeout         %d sec\n", timeout)

		// max_retry
		maxRetry := m.PC.Hooks.MaxRetry
		if maxRetry == 0 {
			maxRetry = 3
		}
		fmt.Printf("  [3] max_retry       %d\n", maxRetry)

		fmt.Println()
		fmt.Println("  [b] Back")
		fmt.Printf("\n%sSelect:%s ", colorCyan, colorReset)

		input := strings.TrimSpace(strings.ToLower(readLine()))

		switch input {
		case "1":
			editor := NewStringSliceEditor("hooks.on_completion", m.PC.Hooks.OnCompletion)
			result, saved, _ := editor.Run()
			if saved {
				m.PC.Hooks.OnCompletion = result
				m.changed = true
			}
		case "2":
			fmt.Printf("Enter timeout (seconds, current: %d): ", timeout)
			numStr := strings.TrimSpace(readLine())
			if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
				m.PC.Hooks.Timeout = num
				m.changed = true
				fmt.Printf("%s✓ timeout = %d%s\n", colorGreen, num, colorReset)
			}
		case "3":
			fmt.Printf("Enter max_retry (current: %d): ", maxRetry)
			numStr := strings.TrimSpace(readLine())
			if num, err := strconv.Atoi(numStr); err == nil && num >= 0 {
				m.PC.Hooks.MaxRetry = num
				m.changed = true
				fmt.Printf("%s✓ max_retry = %d%s\n", colorGreen, num, colorReset)
			}
		case "b", "back":
			// hooks が全て空なら nil に戻す
			if len(m.PC.Hooks.OnCompletion) == 0 && m.PC.Hooks.Timeout == 0 && m.PC.Hooks.MaxRetry == 0 {
				m.PC.Hooks = nil
			}
			return
		default:
			fmt.Printf("%sUnknown command. Use 1-3/b%s\n", colorDim, colorReset)
		}
	}
}

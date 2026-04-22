package ui

import (
	"fmt"
	"strings"
)

// showContext は Context を表示する（編集不可）。
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

// editRules は Rules を StringSliceEditor で編集する。
func (m *ProjectMenu) editRules() {
	editor := NewStringSliceEditorWithRuntime("rules", m.PC.Rules, m.Runtime)
	result, saved, _ := editor.Run()
	if !saved {
		return
	}
	m.PC.Rules = result
	m.changed = true
}

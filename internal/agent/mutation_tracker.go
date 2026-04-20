package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
)

type MutationTracker struct {
	agent *Agent
}

func newMutationTracker(agent *Agent) *MutationTracker {
	return &MutationTracker{agent: agent}
}

func (a *Agent) mutationTracker() *MutationTracker {
	return newMutationTracker(a)
}

func (m *MutationTracker) RecordToolResult(tc *tools.ToolCall, result string, change *tools.FileChange, turnMutations *turnMutationState) {
	if m == nil || m.agent == nil {
		return
	}
	m.NoteProjectMapMutation(tc, change)
	m.trackDeferredDiagnostics(tc, result, change)
	m.RecordFileChangeForTurn(change, turnMutations)
}

func (m *MutationTracker) NoteProjectMapMutation(tc *tools.ToolCall, change *tools.FileChange) {
	a := m.agent
	if a == nil {
		return
	}
	if change != nil {
		a.promptManager().InvalidateProjectMap()
		return
	}
	if tc == nil {
		return
	}
	if tc.Tool == "bash" {
		if !tools.IsReadOnlyBashCommand(tc.Args["command"]) {
			a.promptManager().InvalidateProjectMap()
		}
		return
	}
	if tools.IsWriteTool(tc.Tool) {
		a.promptManager().InvalidateProjectMap()
	}
}

func (m *MutationTracker) RecordFileChange(change *tools.FileChange) {
	m.RecordFileChangeForTurn(change, nil)
}

func (m *MutationTracker) RecordFileChangeForTurn(change *tools.FileChange, turnMutations *turnMutationState) {
	a := m.agent
	if a == nil || change == nil {
		return
	}

	a.promptManager().InvalidateProjectMap()

	a.appendChange(*change)
	if turnMutations != nil {
		turnMutations.recordFileChange(*change)
	}

	if a.changeStorage != nil && a.session != nil {
		if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to persist change: %v\n", err)
		}
	}
}

func (m *MutationTracker) AddPendingLSPFile(path string) {
	a := m.agent
	if a == nil || path == "" {
		return
	}
	for _, f := range a.pendingLSPFiles {
		if f == path {
			return
		}
	}
	a.pendingLSPFiles = append(a.pendingLSPFiles, path)
}

func (m *MutationTracker) AddPendingLSPFilesFromChange(change *tools.FileChange) {
	if change == nil {
		return
	}
	for _, d := range change.Details {
		m.AddPendingLSPFile(d.FilePath)
	}
}

// addPendingLSPFile は編集ツール成功後に対象ファイルを遅延診断バッファへ追加する。
// 重複ファイルは追加しない（連続編集で同一ファイルを複数回編集した場合も1エントリ）。
func (a *Agent) addPendingLSPFile(path string) {
	a.mutationTracker().AddPendingLSPFile(path)
}

// addPendingLSPFilesFromChange は FileChange 内の全ファイルを遅延診断バッファへ追加する。
// apply_patch のように複数ファイルを一度に変更するツール向け。
func (a *Agent) addPendingLSPFilesFromChange(change *tools.FileChange) {
	a.mutationTracker().AddPendingLSPFilesFromChange(change)
}

func (m *MutationTracker) FlushDeferredDiagnostics() string {
	a := m.agent
	if a == nil || len(a.pendingLSPFiles) == 0 {
		return ""
	}
	files := a.pendingLSPFiles
	a.pendingLSPFiles = nil

	result := toolslsp.CheckDiagnosticsForFilesWithClient(a.GetLSPClient(), files)
	if result.Summary == "" {
		return ""
	}
	return "\n\n⚠️ LSP Diagnostics (deferred):\n" + result.Summary
}

func (m *MutationTracker) trackDeferredDiagnostics(tc *tools.ToolCall, result string, change *tools.FileChange) {
	if tc == nil {
		return
	}
	if strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "[CANCELLED]") ||
		strings.HasPrefix(result, "[COMMENT]") {
		return
	}

	switch tc.Tool {
	case "str_replace":
		m.AddPendingLSPFile(tc.Args["path"])
	case "apply_patch":
		m.AddPendingLSPFilesFromChange(change)
	}
}

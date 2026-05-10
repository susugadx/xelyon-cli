package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
	toolslsp "github.com/susugadx/xelyon-cli/internal/tools/lsp"
)

type MutationTracker struct {
	agent *Agent
}

type toolResultRecord struct {
	toolCall                *tools.ToolCall
	result                  string
	change                  *tools.FileChange
	observation             *tools.RuntimeObservation
	turnMutations           *turnMutationState
	trackProjectMapMutation bool
}

func newMutationTracker(agent *Agent) *MutationTracker {
	return &MutationTracker{agent: agent}
}

func (a *Agent) mutationTracker() *MutationTracker {
	return newMutationTracker(a)
}

func (m *MutationTracker) RecordToolResult(tc *tools.ToolCall, result string, change *tools.FileChange, turnMutations *turnMutationState) {
	m.recordToolResult(toolResultRecord{
		toolCall:                tc,
		result:                  result,
		change:                  change,
		turnMutations:           turnMutations,
		trackProjectMapMutation: true,
	})
}

func (m *MutationTracker) RecordToolExecutionResult(tc *tools.ToolCall, result string, change *tools.FileChange, observation *tools.RuntimeObservation, turnMutations *turnMutationState) {
	m.recordToolResult(toolResultRecord{
		toolCall:                tc,
		result:                  result,
		change:                  change,
		observation:             observation,
		turnMutations:           turnMutations,
		trackProjectMapMutation: true,
	})
}

func (m *MutationTracker) recordToolResult(record toolResultRecord) {
	if m == nil || m.agent == nil {
		return
	}
	m.recordLedgerToolObservation(record)
	if record.trackProjectMapMutation {
		m.NoteProjectMapMutation(record.toolCall, record.change)
	}
	m.trackDeferredDiagnostics(record.toolCall, record.result, record.change)
	m.RecordFileChangeForTurn(record.change, record.turnMutations)
}

func (m *MutationTracker) recordLedgerToolObservation(record toolResultRecord) {
	a := m.agent
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return
	}
	observation := ledger.ToolObservation{
		InvocationCWD: a.Runtime.InvocationCWD,
		Result:        record.result,
		Change:        record.change,
		Structured:    record.observation,
		Error:         tools.IsErrorResult(record.result),
	}
	if record.toolCall != nil {
		observation.ToolName = record.toolCall.Tool
		observation.ToolCallID = record.toolCall.ID
		observation.Args = record.toolCall.Args
	} else if record.change != nil {
		observation.ToolName = record.change.Tool
	}
	a.Runtime.TaskLedger.Recorder().RecordToolObservation(observation)
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
		if err := a.changeStorage.AppendChange(a.session.ID, toHistoryChangeRecordInput(*change)); err != nil {
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
		m.AddPendingLSPFile(preferredChangedFilePath(change, tc))
	case "apply_patch":
		m.AddPendingLSPFilesFromChange(change)
	}
}

func preferredChangedFilePath(change *tools.FileChange, tc *tools.ToolCall) string {
	if change != nil {
		for _, detail := range change.Details {
			if strings.TrimSpace(detail.FilePath) != "" {
				return detail.FilePath
			}
		}
		if strings.TrimSpace(change.FilePath) != "" {
			return change.FilePath
		}
	}
	if tc == nil {
		return ""
	}
	return tc.Args["path"]
}

func toHistoryChangeRecordInput(change tools.FileChange) history.ChangeRecordInput {
	details := make([]history.ChangeDetail, 0, len(change.Details))
	for _, detail := range change.Details {
		details = append(details, history.ChangeDetail{FilePath: detail.FilePath})
	}
	return history.ChangeRecordInput{
		FilePath:    change.FilePath,
		Details:     details,
		Timestamp:   change.Timestamp,
		Tool:        change.Tool,
		Description: change.Description,
	}
}

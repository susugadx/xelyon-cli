package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
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

func (m *MutationTracker) RecordToolResult(tc *tools.ToolCall, result string, change *tools.FileChange) {
	if m == nil || m.agent == nil {
		return
	}
	m.NoteProjectMapMutation(tc, change)
	m.trackDeferredDiagnostics(tc, result, change)
	m.RecordFileChange(change)
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
	a := m.agent
	if a == nil || change == nil {
		return
	}

	a.promptManager().InvalidateProjectMap()

	a.changeStack = append(a.changeStack, *change)
	if len(a.changeStack) > config.MaxChangeStack {
		a.changeStack = a.changeStack[1:]
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

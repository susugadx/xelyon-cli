package tui

import (
	"strings"
	"testing"
	"time"
)

func TestModel_UpdateStatusMsgResetsStreamingState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.streamingActive = true
	m.streamCursorCol = 7
	m.streamActiveANSI = "\033[31m"
	m.streamPendingANSI = "\033[3"

	updated, _, handled := m.handleStreamMessage(UpdateStatusMsg{Line: "done"})
	if !handled {
		t.Fatal("UpdateStatusMsg should be handled by stream message handler")
	}
	m = updated

	if m.streamingActive {
		t.Fatal("streamingActive should be reset")
	}
	if m.streamCursorCol != 0 || m.streamActiveANSI != "" || m.streamPendingANSI != "" {
		t.Fatalf("stream state = (%d,%q,%q), want zero values", m.streamCursorCol, m.streamActiveANSI, m.streamPendingANSI)
	}
	if m.statusLine != "done" {
		t.Fatalf("statusLine = %q, want %q", m.statusLine, "done")
	}
}

func TestModel_UpdateStatusMsgUpdatesSnapshotMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.statusSnapshot = StatusSnapshot{
		Mode:       "Plan",
		LegacyLine: "ready",
	}

	updated, _, handled := m.handleStreamMessage(UpdateStatusMsg{Line: "running"})
	if !handled {
		t.Fatal("UpdateStatusMsg should be handled by stream message handler")
	}
	m = updated

	plain := stripANSI(m.buildStatusText(time.Now()))
	if !strings.Contains(plain, "running") {
		t.Fatalf("status text should contain updated status, got %q", plain)
	}
	if strings.Contains(plain, "Plan") {
		t.Fatalf("status text should not keep stale mode, got %q", plain)
	}
}

func TestModel_AppendToolResultMsgResetsStreamingState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.streamingActive = true
	m.streamCursorCol = 3
	m.streamActiveANSI = "\033[32m"
	m.streamPendingANSI = "\033[3"

	updated, _, handled := m.handleStreamMessage(AppendToolResultMsg{
		Tool: ToolResult{Name: "read_file", Summary: "read_file", Detail: "detail", Collapsed: true},
	})
	if !handled {
		t.Fatal("AppendToolResultMsg should be handled by stream message handler")
	}
	m = updated

	if m.streamingActive {
		t.Fatal("streamingActive should be reset")
	}
	if m.streamCursorCol != 0 || m.streamActiveANSI != "" || m.streamPendingANSI != "" {
		t.Fatalf("stream state = (%d,%q,%q), want zero values", m.streamCursorCol, m.streamActiveANSI, m.streamPendingANSI)
	}
	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
}

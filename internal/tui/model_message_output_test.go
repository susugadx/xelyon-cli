package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_AppendMessageKeepsViewportAtBottom(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 8})
	m = updated.(Model)

	for i := 0; i < 8; i++ {
		updated, _ = m.Update(AppendMessageMsg{
			Message: ChatMessage{Role: "assistant", Content: "line"},
		})
		m = updated.(Model)
	}

	if !m.vp.atBottom() {
		t.Fatal("viewport should stay pinned to bottom after appends")
	}
}

func TestModel_AppendMessagePreservesCRLFSeparatedLines(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "first\r\nsecond\r\nthird"},
	})
	m = updated.(Model)

	if len(m.rawLines) != 4 {
		t.Fatalf("rawLines len = %d, want 4", len(m.rawLines))
	}
	want := []string{"│ first", "│ second", "│ third"}
	for i, line := range want {
		if m.rawLines[i+1] != line {
			t.Fatalf("rawLines[%d] = %q, want %q", i+1, m.rawLines[i+1], line)
		}
	}
	if strings.Contains(m.viewportView(), "\r") {
		t.Fatalf("viewportView should not contain carriage return, got %q", m.viewportView())
	}
}

func TestModel_AppendAssistantChunkKeepsRawLinesWithoutTurnChrome(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	m = updated.(Model)

	for _, text := range []string{"first", "second"} {
		updated, _ = m.Update(AppendMessageMsg{
			Message: ChatMessage{Role: ChatRoleAssistantChunk, Content: text},
		})
		m = updated.(Model)
	}

	want := []string{"first", "second"}
	if len(m.rawLines) != len(want) {
		t.Fatalf("rawLines len = %d, want %d: %#v", len(m.rawLines), len(want), m.rawLines)
	}
	for i := range want {
		if m.rawLines[i] != want[i] {
			t.Fatalf("rawLines[%d] = %q, want %q", i, m.rawLines[i], want[i])
		}
	}
}

func TestModel_AppendMessageFillsZeroTimestamp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "hello"},
	})
	m = updated.(Model)

	if len(m.messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(m.messages))
	}
	if m.messages[0].Timestamp.IsZero() {
		t.Fatal("appendMessage should fill zero timestamp")
	}
	if got := m.rawLines[0]; !strings.Contains(got, m.messages[0].Timestamp.Format("15:04")) {
		t.Fatalf("separator line = %q, want filled timestamp", got)
	}
}

func TestModel_AppendMessagePreservesTabsInRawLines(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "a\tb"},
	})
	m = updated.(Model)

	if len(m.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.rawLines))
	}
	if m.rawLines[1] != "│ a\tb" {
		t.Fatalf("rawLines[1] = %q, want %q", m.rawLines[1], "│ a\tb")
	}
	if got := stripANSI(m.viewportView()); !strings.Contains(got, "a    b") {
		t.Fatalf("viewportView should expand tabs for display, got %q", got)
	}
}

func TestModel_CopyRawRangePreservesTabs(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 18, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "line1\tvalue"},
	})
	m = updated.(Model)

	if err := m.copyRawRangePlain(1, 1); err != nil {
		t.Fatalf("copyRawRangePlain() error = %v", err)
	}
	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "│ line1\tvalue" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "│ line1\tvalue")
	}
}

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIIntegration_StreamResizeKeepsANSIWrappedViewport(t *testing.T) {
	m, _ := newStreamTestModel(12, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mabcdefghij\033[0m\n\tjk", Done: true})
	m = updated.(Model)

	rowsBefore := len(m.layout.Rows)
	beforeView := m.viewportView()
	if !strings.Contains(beforeView, "\033[31m") {
		t.Fatalf("viewportView should preserve ANSI before resize, got %q", beforeView)
	}
	if plain := stripANSI(beforeView); !strings.Contains(plain, "abcdefghij") || !strings.Contains(plain, "    jk") {
		t.Fatalf("viewportView before resize should contain wrapped text and expanded tab, got %q", plain)
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 6, Height: 8})
	m = updated.(Model)

	afterView := m.viewportView()
	if len(m.layout.Rows) <= rowsBefore {
		t.Fatalf("layout rows after resize = %d, want > %d", len(m.layout.Rows), rowsBefore)
	}
	if strings.Contains(afterView, "\r") {
		t.Fatalf("viewportView should not contain carriage return after resize, got %q", afterView)
	}
	if !strings.Contains(afterView, "\033[31m") {
		t.Fatalf("viewportView should preserve ANSI after resize, got %q", afterView)
	}
	if plain := stripANSI(afterView); !strings.Contains(plain, "abcd") || !strings.Contains(plain, "ghij") || !strings.Contains(plain, "    jk") {
		t.Fatalf("viewportView after resize should contain wrapped text and expanded tab, got %q", plain)
	}
}

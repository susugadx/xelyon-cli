package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestModel_RebuildChromePadsInputAreaToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "\033[33mready\033[0m"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 18
	m.height = 8
	m.vp = lightViewport{width: 18, height: 4}
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	m.textInput.Width = max(0, m.width-inputPromptWidth-1)
	m.textInput.SetValue("日本a")
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4", len(lines))
	}

	for i := 0; i < 3; i++ {
		if got := lipgloss.Width(lines[i]); got != m.width {
			t.Fatalf("chrome line %d width = %d, want %d; line=%q", i, got, m.width, lines[i])
		}
	}
	if got := lipgloss.Width(lines[3]); got != m.width {
		t.Fatalf("status line width = %d, want %d; line=%q", got, m.width, lines[3])
	}
	if !strings.Contains(lines[1], inputPrompt) {
		t.Fatalf("input line should contain prompt, got %q", lines[1])
	}
	if stripANSI(lines[0]) != stripANSI(lines[2]) {
		t.Fatalf("top/bottom padding should match, got %q / %q", lines[0], lines[2])
	}
}

func TestModel_RebuildChromeKeepsInputAreaAlignedAtNarrowWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 6
	m.height = 8
	m.vp = lightViewport{width: 6, height: 4}
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	m.textInput.Width = max(0, m.width-inputPromptWidth-1)
	m.textInput.SetValue("abcdef")
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4", len(lines))
	}
	for i := 0; i < 3; i++ {
		if got := lipgloss.Width(lines[i]); got != m.width {
			t.Fatalf("chrome line %d width = %d, want %d; line=%q", i, got, m.width, lines[i])
		}
	}
	if got := stripANSI(lines[1]); len(got) == 0 {
		t.Fatalf("input line should keep visible content, got %q", lines[1])
	}
}

func TestModel_RebuildChromeTruncatesLongANSIInputToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "\033[36mstatus\033[0m"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 12
	m.height = 8
	m.vp = lightViewport{width: 12, height: 4}
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	m.textInput.Width = max(0, m.width-inputPromptWidth-1)
	m.textInput.SetValue("日本語abcdef")
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4", len(lines))
	}
	if got := lipgloss.Width(lines[1]); got != m.width {
		t.Fatalf("input line width = %d, want %d; line=%q", got, m.width, lines[1])
	}
	if !strings.HasSuffix(lines[1], "\033[0m") {
		t.Fatalf("input line should end with reset, got %q", lines[1])
	}
	if got := lipgloss.Width(lines[3]); got != m.width {
		t.Fatalf("status line width = %d, want %d; line=%q", got, m.width, lines[3])
	}
}

func TestModel_RenderInputDock_StructureAndPrompt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("hello")
	m.rebuildChrome()

	dock := m.renderInputDock()
	lines := strings.Split(dock, "\n")
	if len(lines) != 3 {
		t.Fatalf("renderInputDock should produce 3 lines (pad+input+pad), got %d", len(lines))
	}
	if !strings.Contains(lines[1], inputPrompt) {
		t.Fatalf("input line should contain prompt %q, got %q", inputPrompt, lines[1])
	}
	if !strings.Contains(lines[1], "hello") {
		t.Fatalf("input line should contain input value, got %q", lines[1])
	}
}

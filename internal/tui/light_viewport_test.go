package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestLightViewport_ViewPadsRowsToWidth(t *testing.T) {
	v := lightViewport{
		width:  8,
		height: 3,
		lines: []string{
			"\033[31mabc\033[0m",
			"日本a",
		},
	}

	lines := strings.Split(v.view(), "\n")
	if len(lines) != 3 {
		t.Fatalf("view lines = %d, want 3", len(lines))
	}

	for i, line := range lines {
		if got := lipgloss.Width(line); got != v.width {
			t.Fatalf("line %d width = %d, want %d; line=%q", i, got, v.width, line)
		}
	}
}

func TestLightViewport_ViewClearsLongLineFragmentsAfterScroll(t *testing.T) {
	v := lightViewport{
		width:  10,
		height: 2,
		lines: []string{
			"0123456789",
			"tiny",
			"\033[32m日本語\033[0m",
		},
	}

	before := strings.Split(v.view(), "\n")
	if got := stripANSI(before[0]); got != "0123456789" {
		t.Fatalf("before scroll first row = %q, want %q", got, "0123456789")
	}

	v.scrollDown(1)
	after := strings.Split(v.view(), "\n")
	if len(after) != v.height {
		t.Fatalf("view lines = %d, want %d", len(after), v.height)
	}
	if got := stripANSI(after[0]); got != "tiny      " {
		t.Fatalf("after scroll first row = %q, want %q", got, "tiny      ")
	}
	if got := lipgloss.Width(after[0]); got != v.width {
		t.Fatalf("after scroll first row width = %d, want %d; line=%q", got, v.width, after[0])
	}
	if got := lipgloss.Width(after[1]); got != v.width {
		t.Fatalf("after scroll second row width = %d, want %d; line=%q", got, v.width, after[1])
	}
}

func TestModel_ViewPadsScrolledRowsToViewportWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 8})
	m = updated.(Model)

	m.appendContentLines(
		"0123456789",
		"shrt",
		"\033[32mgo test ./...\033[0m",
		"日本語mix",
	)
	m.vp.gotoTop()

	body := strings.Split(m.viewportView(), "\n")
	if len(body) != m.vp.height {
		t.Fatalf("body lines = %d, want %d", len(body), m.vp.height)
	}

	for i, line := range body {
		if got := lipgloss.Width(line); got != m.vp.width {
			t.Fatalf("body line %d width = %d, want %d; line=%q", i, got, m.vp.width, line)
		}
	}
}

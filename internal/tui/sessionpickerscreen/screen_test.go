package sessionpickerscreen

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func TestHandleKeyFiltersAndReturnsSelectedSession(t *testing.T) {
	screen := New([]Candidate{
		{ID: "session-a", Preview: "first", WorkingDir: "/repo/a"},
		{ID: "session-b", Preview: "target preview", WorkingDir: "/repo/b"},
	}, false, true)

	screen.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	screen.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("target")})
	result := screen.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if result.Command != CommandResume {
		t.Fatalf("command = %d, want CommandResume", result.Command)
	}
	if result.Candidate.ID != "session-b" {
		t.Fatalf("candidate ID = %q, want session-b", result.Candidate.ID)
	}
	if snapshot := screen.Snapshot(); !snapshot.Startup || !snapshot.Filtering || snapshot.Filter != "target" {
		t.Fatalf("snapshot = %#v, want startup filtered target", snapshot)
	}
}

func TestPanelLinesShowsAllSessionsTitleAndSanitizedLabel(t *testing.T) {
	screen := New([]Candidate{{
		ID:           "session-123456789",
		Preview:      "first\nsecond",
		ProviderName: "openai",
		Model:        "gpt-test",
		WorkingDir:   "/tmp/project",
		LastModified: time.Date(2026, time.June, 18, 12, 0, 0, 0, time.UTC),
	}}, true, false)

	plain := termtext.StripANSI(strings.Join(screen.PanelLines(80, 24), "\n"))
	for _, want := range []string{
		"Resume - all sessions",
		"2026-06-18 12:00",
		"session-",
		"openai/gpt-test",
		"project",
		"first second",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("panel missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "\nsecond") {
		t.Fatalf("panel contains unsanitized preview newline:\n%s", plain)
	}
}

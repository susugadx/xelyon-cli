package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestModel_StatusBarClampedToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: strings.Repeat("status ", 20)}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.vp = lightViewport{width: 20, height: 4}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4", len(lines))
	}

	statusLine := stripANSI(lines[len(lines)-1])
	if got := lipgloss.Width(statusLine); got != m.width {
		t.Fatalf("status line width = %d, want %d; line=%q", got, m.width, statusLine)
	}
}

func TestModel_RebuildChromeSanitizesMultilineStatusAndTransient(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 40
	m.height = 8
	m.vp = lightViewport{width: 40, height: 4}
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	m.statusLine = "phase1\rphase2\nphase3\tok"
	m.transientStatus = "copy\nok\r!"
	m.transientStatusUntil = time.Now().Add(1 * time.Second)
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4; cache=%q", len(lines), m.chromeCache)
	}
	if got := lipgloss.Width(lines[3]); got != m.width {
		t.Fatalf("status line width = %d, want %d; line=%q", got, m.width, lines[3])
	}
	plain := stripANSI(lines[3])
	if !strings.Contains(plain, "phase1 phase2 phase3 ok") {
		t.Fatalf("status line should be flattened to single line, got %q", plain)
	}
	if strings.ContainsAny(plain, "\r\n\t") {
		t.Fatalf("status line should not contain control line-break chars, got %q", plain)
	}
}

func TestModel_RenderStatusBar_ContainsHints(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	bar := m.renderStatusBar()
	lines := strings.Split(bar, "\n")
	if len(lines) != 1 {
		t.Fatalf("renderStatusBar should produce 1 line, got %d", len(lines))
	}
	if !strings.Contains(bar, "ready") {
		t.Fatalf("status bar should contain status text, got %q", bar)
	}
}

func TestModel_RenderStatusBar_ShowsWorkingDirWhenSpaceAllows(t *testing.T) {
	t.Setenv("HOME", filepath.Join(string(filepath.Separator), "tmp", "xelyon-test-home"))

	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.width = 80
	m.workingDir = filepath.Join(string(filepath.Separator), "opt", "xelyon", "dev", "xelyon-cli")

	bar := stripANSI(m.renderStatusBar())
	if !strings.Contains(bar, "cwd: /opt/xelyon/dev/xelyon-cli") {
		t.Fatalf("status bar should contain working dir, got %q", bar)
	}
	if !strings.Contains(bar, "Esc:NAV") {
		t.Fatalf("status bar should preserve hints, got %q", bar)
	}
}

func TestModel_RenderStatusBar_HidesWorkingDirWhenNarrow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.width = 24
	m.workingDir = filepath.Join(string(filepath.Separator), "opt", "xelyon", "dev", "xelyon-cli")

	bar := stripANSI(m.renderStatusBar())
	if strings.Contains(bar, "cwd:") {
		t.Fatalf("status bar should hide working dir when narrow, got %q", bar)
	}
	if !strings.Contains(bar, "Esc:NAV") {
		t.Fatalf("status bar should keep fitting hint, got %q", bar)
	}
}

func TestModel_RenderStatusBar_NavMode(t *testing.T) {
	agent := &stubAgent{statusLine: "nav status"}
	m := newModelWithViewport(agent)
	m.navigationMode = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "NAV") {
		t.Fatalf("nav mode status bar should contain NAV badge, got %q", bar)
	}
}

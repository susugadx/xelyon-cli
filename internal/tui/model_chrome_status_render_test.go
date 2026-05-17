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
	if !strings.Contains(bar, "/plan") {
		t.Fatalf("status bar should show plan-mode command hint, got %q", bar)
	}
}

func TestModel_BuildStatusTextPrioritizesProcessingSegments(t *testing.T) {
	agent := &stubAgent{statusLine: "ready", processing: true}
	m := newModelWithViewport(agent)
	m.statusLine = "phase1\rphase2\nphase3\tok"
	m.newOutput = true
	m.transientStatus = "copy\nok\r!"
	now := time.Now()
	m.transientStatusUntil = now.Add(time.Minute)
	m.vp = lightViewport{
		lines:   []string{"one", "two", "three"},
		yOffset: 0,
		width:   m.width,
		height:  1,
	}

	plain := stripANSI(m.buildStatusText(now))
	for _, fragment := range []string{"phase1 phase2 phase3 ok"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("status text missing %q, got %q", fragment, plain)
		}
	}
	for _, fragment := range []string{"New output", "copy ok !"} {
		if strings.Contains(plain, fragment) {
			t.Fatalf("processing status should defer %q, got %q", fragment, plain)
		}
	}
	if strings.ContainsAny(plain, "\r\n\t") {
		t.Fatalf("status text should not contain control line-break chars, got %q", plain)
	}
}

func TestModel_BuildStatusTextKeepsProcessingSummaryCompact(t *testing.T) {
	agent := &stubAgent{
		statusLine: "legacy",
		processing: true,
		statusSnapshot: StatusSnapshot{
			Provider:   "openai",
			Model:      "gpt-5.4",
			Mode:       "Plan: ON",
			Tokens:     "12.3k",
			Cost:       "~$0.123",
			LegacyLine: "legacy",
		},
	}
	m := newModelWithViewport(agent)
	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "● running read_file internal/tui/model.go",
		Target:    "internal/tui/model.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})

	plain := stripANSI(m.buildStatusText(time.Now()))
	for _, fragment := range []string{"openai/gpt-5.4", "Plan: ON", "12.3k tok", "~$0.123"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("status text missing %q, got %q", fragment, plain)
		}
	}
	if strings.Contains(plain, "running read_file") {
		t.Fatalf("processing status should not include running tool detail, got %q", plain)
	}
}

func TestModel_BuildStatusTextIdleUsesLegacyLineDetails(t *testing.T) {
	statusLine := "● gpt-5.4 │ openai │ Plan: OFF │ 12.3k │ ~$0.123"
	agent := &stubAgent{
		statusLine: statusLine,
		statusSnapshot: StatusSnapshot{
			Provider:   "openai",
			Model:      "gpt-5.4",
			Mode:       "Plan: OFF",
			Tokens:     "12.3k",
			Cost:       "~$0.123",
			LegacyLine: statusLine,
		},
	}
	m := newModelWithViewport(agent)

	plain := stripANSI(m.buildStatusText(time.Now()))
	for _, fragment := range []string{"gpt-5.4", "openai", "Plan: OFF", "12.3k", "~$0.123"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("idle status text missing %q, got %q", fragment, plain)
		}
	}
	if strings.TrimSpace(plain) == "Plan: OFF" {
		t.Fatalf("idle status text should not collapse to mode only, got %q", plain)
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

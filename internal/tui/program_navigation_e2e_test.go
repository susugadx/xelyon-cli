package tui

import (
	"strings"
	"testing"
	"time"
)

func TestTUIProgramE2E_NavigationVisualSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")
	model.rawLines = []string{
		"alpha beta gamma",
		"second line",
		"third line",
		"fourth line",
		"fifth line",
		"sixth line",
	}

	final := setup(8, 12).
		nav().
		build().
		appendScript(keys("gg")).
		appendScript(keys("wvj")).
		appendScript(keys("<WheelDown>")).
		append(sendMsg(resize(14, 10))).
		run(t, model)

	if !final.ready {
		t.Fatal("model should be ready after window sizing")
	}
	if !final.navigationMode {
		t.Fatal("navigationMode should remain enabled")
	}
	if final.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", final.visualMode, visualModeChar)
	}
	if final.visualStart != (visualPosition{line: 0, col: 6}) {
		t.Fatalf("visualStart = %+v, want {line:0 col:6}", final.visualStart)
	}
	if final.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", final.cursorLine)
	}
	if final.cursorCol != 6 {
		t.Fatalf("cursorCol = %d, want 6", final.cursorCol)
	}
	lines := normalizedViewLines(final.View())
	if len(lines) < 6 {
		t.Fatalf("view lines = %d, want >= 6", len(lines))
	}
	assertGoldenText(t, "navigation_visual_body", strings.Join(lines[:2], "\n"))
}

func TestTUIProgramE2E_CopySetsTransientStatus(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")
	model.rawLines = []string{"copy target"}

	final := setup(20, 8).
		nav().
		build().
		appendScript(keys("yy")).
		run(t, model)

	if final.transientStatus == "" {
		t.Fatal("transientStatus should be set after yy copy")
	}
	if !strings.Contains(final.transientStatus, "Copied") {
		t.Fatalf("transientStatus = %q, want copy message", final.transientStatus)
	}
	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "copy target" {
		t.Fatalf("copyTexts = %#v, want [copy target]", agent.copyTexts)
	}
	if !strings.Contains(stripANSI(final.chromeCache), "Cop") {
		t.Fatalf("chromeCache should include truncated transient copy status, got %q", final.chromeCache)
	}
	lines := normalizedViewLines(final.View())
	if len(lines) < 4 {
		t.Fatalf("view lines = %d, want >= 4", len(lines))
	}
	assertGoldenText(t, "copy_transient_footer", lines[len(lines)-1])
}

func TestTUIProgramE2E_CtrlCTwiceQuits(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(20, 8).
		build().
		appendScript(keys("<Ctrl+C><Ctrl+C>")).
		run(t, model)

	if !final.quitting {
		t.Fatal("model should be in quitting state after second ctrl+c")
	}
	if agent.cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", agent.cleanupCalls)
	}
	if final.lastInterrupt.IsZero() {
		t.Fatal("lastInterrupt should be recorded")
	}
}

func TestTUIProgramE2E_CtrlCTimeoutDoesNotQuit(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(20, 8).
		build().
		appendScript(keys("<Ctrl+C>")).
		append(sendMsg(UpdateStatusMsg{Line: "still-running"})).
		run(t, model)
	final.lastInterrupt = time.Now().Add(-4 * time.Second)

	if final.quitting {
		t.Fatal("model should not quit after single ctrl+c")
	}
	if agent.cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", agent.cleanupCalls)
	}
}

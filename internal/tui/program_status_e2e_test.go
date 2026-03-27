package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIProgramE2E_NewOutputBadgeLifecycle(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(12, 9).
		assistantLines("l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9", "l10").
		build().
		appendScript(keys("<PgUp>")).
		append(appendAssistant("new output")).
		appendScript(keys("<PgDown><PgDown>")).
		run(t, model)

	if final.newOutput {
		t.Fatal("newOutput should be cleared after returning to bottom")
	}
	if strings.Contains(final.chromeCache, "New output") {
		t.Fatalf("chromeCache should not contain badge after returning to bottom, got %q", final.chromeCache)
	}
	if !final.vp.atBottom() {
		t.Fatal("viewport should end at bottom")
	}
}

func TestTUIProgramE2E_StreamStatusSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(20, 8).
		status("processing").
		build().
		append(sendMsg(StreamTextMsg{Text: "hello", Done: false}), sendMsg(StreamTextMsg{Text: "\nworld", Done: true})).
		run(t, model)

	if len(final.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(final.rawLines))
	}
	if final.rawLines[0] != "hello" || final.rawLines[1] != "world" {
		t.Fatalf("rawLines = %#v, want [hello world]", final.rawLines)
	}
	if final.statusLine != "ready" {
		t.Fatalf("statusLine = %q, want %q", final.statusLine, "ready")
	}
	lines := normalizedViewLines(final.View())
	if len(lines) < 6 {
		t.Fatalf("view lines = %d, want >= 6", len(lines))
	}
	assertGoldenText(t, "stream_status_body", strings.Join(lines[:2], "\n"))
	assertGoldenText(t, "stream_status_footer", lines[len(lines)-1])
}

func TestTUIProgramE2E_SpinnerTickAndAgentDoneRefreshStatus(t *testing.T) {
	agent := &stubAgent{
		statusLine: "processing",
		processing: true,
	}
	model := NewModel(agent, "")

	final := script(
		sendMsg(resize(24, 8)),
		sendMsg(spinner.TickMsg{}),
		func(p *tea.Program) {
			agent.setProcessing(false)
			agent.setStatusLine("ready")
			p.Send(AgentDoneMsg{})
		},
	).run(t, model)

	if final.statusLine != "ready" {
		t.Fatalf("statusLine = %q, want %q", final.statusLine, "ready")
	}
	if !strings.Contains(stripANSI(final.chromeCache), "ready") {
		t.Fatalf("chromeCache should contain final ready status, got %q", final.chromeCache)
	}
	if strings.Contains(stripANSI(final.chromeCache), "processing") {
		t.Fatalf("chromeCache should not contain stale processing status, got %q", final.chromeCache)
	}
	lines := normalizedViewLines(final.View())
	if len(lines) == 0 {
		t.Fatal("view should not be empty")
	}
	assertGoldenText(t, "spinner_done_footer", lines[len(lines)-1])
}

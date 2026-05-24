package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReviewCommand_WithInstructionsKeepsDraftWhenBusy(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{
			statusLine: "processing",
			processing: true,
		},
		report: newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/review focus on regressions")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/review busy cmd = %v, want nil", cmd)
	}
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 while agent is processing", agent.reviewCalls)
	}
	if got := m.textInput.Value(); got != "/review focus on regressions" {
		t.Fatalf("composer draft = %q, want original command while busy", got)
	}
	if m.transientStatus != agentTurnBusyStatus {
		t.Fatalf("transientStatus = %q, want %q", m.transientStatus, agentTurnBusyStatus)
	}
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should not be created while agent is processing")
	}
}

func TestReviewCommand_WithInstructionsRunsCurrentChangesReview(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/review focus on regressions")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat", m.screen)
	}
	if m.reviewScreen != nil {
		t.Fatal("reviewScreen should remain nil for /review with inline focus")
	}
	if cmd == nil {
		t.Fatal("/review with instructions cmd = nil, want async review command")
	}
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 before cmd execution", agent.reviewCalls)
	}

	for _, msg := range runReviewCommandForTest(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	if agent.reviewCalls != 1 {
		t.Fatalf("RunReview calls = %d, want 1 after cmd execution", agent.reviewCalls)
	}
	if got := agent.lastRequest.CustomInstructions; got != "focus on regressions" {
		t.Fatalf("RunReview CustomInstructions = %q, want focus on regressions", got)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "focus: focus on regressions") {
		t.Fatalf("View() missing inline focus display:\n%s", view)
	}
}

func TestReviewCommand_WithQuotedInstructionsKeepsRawRemainder(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)

	m.textInput.SetValue(`/review "focus on regressions"`)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("/review with quoted instructions cmd = nil, want async review command")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("reviewTimelineRun = nil, want active timeline run")
	}

	for _, msg := range runReviewCommandForTest(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	if got := agent.lastRequest.CustomInstructions; got != `"focus on regressions"` {
		t.Fatalf("RunReview CustomInstructions = %q, want quoted raw instructions", got)
	}
}

func TestReviewCommand_WithUnterminatedQuoteKeepsRawRemainder(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)

	m.textInput.SetValue(`/review investigate "quoted`)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("/review with unterminated quote instructions cmd = nil, want async review command")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("reviewTimelineRun = nil, want active timeline run")
	}

	for _, msg := range runReviewCommandForTest(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	if got := agent.lastRequest.CustomInstructions; got != `investigate "quoted` {
		t.Fatalf("RunReview CustomInstructions = %q, want raw instructions", got)
	}
}

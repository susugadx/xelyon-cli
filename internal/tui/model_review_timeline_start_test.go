package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui/reviewscreen"
)

func TestReviewTimeline_CurrentChangesPresetStartsRun(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newReviewCapableTestModel(agent)

	var cmd tea.Cmd
	m, cmd = sendReviewKeyWithCmd(m, "enter")

	if cmd == nil {
		t.Fatal("current changes preset cmd = nil, want async timeline review command")
	}
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after starting review", m.screen)
	}
	if m.reviewScreen != nil {
		t.Fatal("reviewScreen should close when timeline review starts")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("reviewTimelineRun = nil, want active timeline run")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"/review current changes", "review · working", "running current changes review"} {
		if !strings.Contains(view, want) {
			t.Fatalf("timeline view missing %q:\n%s", want, view)
		}
	}

	m = applyReviewCommandMessages(t, m, cmd)
	if agent.lastRequest.TargetKind != review.TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want current_changes", agent.lastRequest.TargetKind)
	}
	if agent.lastRequest.CustomInstructions != "" {
		t.Fatalf("CustomInstructions = %q, want empty", agent.lastRequest.CustomInstructions)
	}
	view = stripANSI(m.View())
	for _, want := range []string{"review · done", "completed current changes review", "Review result", "Finding: stale result is ignored"} {
		if !strings.Contains(view, want) {
			t.Fatalf("completed timeline view missing %q:\n%s", want, view)
		}
	}
}

func TestReviewTimeline_CustomFocusStartsRun(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newReviewCapableTestModel(agent)

	m = sendReviewKey(m, "down")
	m = sendReviewKey(m, "enter")
	if snapshot := m.reviewScreen.Snapshot(); snapshot.Mode != reviewscreen.ModeCustom {
		t.Fatalf("review mode = %d, want custom", snapshot.Mode)
	}

	m = sendReviewText(m, "focus on regressions")
	var cmd tea.Cmd
	m, cmd = sendReviewKeyWithCmd(m, "enter")

	if cmd == nil {
		t.Fatal("custom focus cmd = nil, want async timeline review command")
	}
	view := stripANSI(m.View())
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after starting custom review", m.screen)
	}
	for _, want := range []string{"/review current changes", "focus: focus on regressions", "review · working"} {
		if !strings.Contains(view, want) {
			t.Fatalf("custom timeline view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Custom instructions:") {
		t.Fatalf("timeline view should not use old custom instructions label:\n%s", view)
	}

	m = applyReviewCommandMessages(t, m, cmd)
	if got := agent.lastRequest.CustomInstructions; got != "focus on regressions" {
		t.Fatalf("RunReview CustomInstructions = %q, want focus on regressions", got)
	}
}

func TestReviewTimeline_NilReviewAgentShowsNotImplemented(t *testing.T) {
	m := newReviewTestModel()

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("startReviewTimeline() cmd = %v, want nil", cmd)
	}
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after review handoff", m.screen)
	}
	if m.reviewScreen != nil {
		t.Fatal("reviewScreen should close before timeline review starts")
	}
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should not be created without a ReviewAgent")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"review · blocked", "[validation error]", reviewRunnerNotImplementedMessage} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestReviewTimeline_BlocksRunWhileAgentIsProcessing(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{
			statusLine: "processing",
			processing: true,
		},
		report: newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = reviewscreen.New(m.width)

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest("focus on regressions"))
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("startReviewTimeline() cmd = %v, want nil while agent is processing", cmd)
	}
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 while agent is processing", agent.reviewCalls)
	}
	if m.reviewTimelineRun != nil {
		t.Fatal("timeline review run should not be created while agent is processing")
	}
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen should remain open when review is rejected as busy")
	}
	if m.transientStatus != agentTurnBusyStatus {
		t.Fatalf("transientStatus = %q, want %q", m.transientStatus, agentTurnBusyStatus)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, agentTurnBusyStatus) {
		t.Fatalf("review screen should show busy notice:\n%s", view)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d, want 1 so in-flight chat remains cancellable", cancelCalls)
	}
	if m.screen != screenReview {
		t.Fatalf("screen = %d after ctrl+c, want screenReview", m.screen)
	}
}

func TestReviewTimeline_BusyRejectionVisibleInCustomFocusScreen(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{
			statusLine: "processing",
			processing: true,
		},
		report: newTUITestReviewReport(),
	}
	m := newReviewCapableTestModel(agent)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)

	m = sendReviewKey(m, "down")
	m = sendReviewKey(m, "enter")
	m = sendReviewText(m, "focus on regressions")
	m, cmd := sendReviewKeyWithCmd(m, "enter")

	if cmd != nil {
		t.Fatalf("custom focus busy cmd = %v, want nil", cmd)
	}
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 while agent is processing", agent.reviewCalls)
	}
	if m.screen != screenReview {
		t.Fatalf("screen = %d, want screenReview after busy rejection", m.screen)
	}
	if m.reviewScreen == nil || m.reviewScreen.Snapshot().Mode != reviewscreen.ModeCustom {
		t.Fatal("custom focus screen should remain open after busy rejection")
	}
	if !m.reviewScreen.Snapshot().CustomInputFocused {
		t.Fatal("custom focus input should remain focused after busy rejection")
	}
	view := stripANSI(m.View())
	for _, want := range []string{agentTurnBusyStatus, "focus on regressions"} {
		if !strings.Contains(view, want) {
			t.Fatalf("custom focus busy view missing %q:\n%s", want, view)
		}
	}
}

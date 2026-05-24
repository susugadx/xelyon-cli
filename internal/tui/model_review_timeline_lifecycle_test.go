package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestReviewTimeline_RunReviewSuccessIsAsyncAndRendered(t *testing.T) {
	agent := &reviewCapableStubAgent{stubAgent: stubAgent{statusLine: "ready"}}
	agent.report = newTUITestReviewReport()
	m := newModelWithViewport(agent)
	if m.reviewAgent == nil {
		t.Fatal("reviewAgent = nil, want optional ReviewAgent to be captured")
	}

	m.screen = screenReview
	m.reviewScreen = newReviewScreen()
	req := review.NewCurrentChangesRequest("")
	updated, cmd := m.startReviewTimeline(req)
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
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
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 before cmd execution", agent.reviewCalls)
	}

	msg := cmd()
	if agent.reviewCalls != 1 {
		t.Fatalf("RunReview calls = %d, want 1 after cmd execution", agent.reviewCalls)
	}
	if agent.lastRequest.TargetKind != req.TargetKind {
		t.Fatalf("last request TargetKind = %q, want %q", agent.lastRequest.TargetKind, req.TargetKind)
	}

	updated, _ = m.Update(msg)
	m = updated.(Model)
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should clear after command result")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"review · done", "Review result", "Verdict: has_findings", "Verification: verified", "Group: request state", "Finding: stale result is ignored"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestReviewTimeline_RefreshesStatusSnapshotAfterCompletion(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{
			statusLine: "ready",
			statusSnapshot: StatusSnapshot{
				Mode:       "ready",
				Tokens:     "10",
				Cost:       "$0.01",
				LegacyLine: "ready",
			},
		},
		report:                newTUITestReviewReport(),
		statusLineAfterReview: "review complete",
		statusSnapshotAfterReview: StatusSnapshot{
			Mode:       "review complete",
			Tokens:     "42",
			Cost:       "$0.42",
			LegacyLine: "review complete",
		},
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
	}
	if got := m.statusSnapshot.Tokens; got != "10" {
		t.Fatalf("initial status tokens = %q, want 10", got)
	}

	m = applyReviewCommandMessages(t, m, cmd)
	if got := m.statusLine; got != "review complete" {
		t.Fatalf("statusLine after review = %q, want review complete", got)
	}
	if got := m.statusSnapshot.Tokens; got != "42" {
		t.Fatalf("status tokens after review = %q, want 42", got)
	}
	if got := m.statusSnapshot.Cost; got != "$0.42" {
		t.Fatalf("status cost after review = %q, want $0.42", got)
	}
	if statusText := stripANSI(m.buildStatusText(time.Now())); !strings.Contains(statusText, "review complete") {
		t.Fatalf("status text missing refreshed status line: %q", statusText)
	}
}

func TestReviewTimeline_CtrlCCancelsRunningReview(t *testing.T) {
	agent := newCancellableReviewAgent()
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen()

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
	}
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after starting review", m.screen)
	}
	if m.reviewScreen != nil {
		t.Fatal("reviewScreen should close when timeline review starts")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("timeline should hold active run while review is running")
	}

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	select {
	case <-agent.started:
	case <-time.After(2 * time.Second):
		t.Fatal("RunReview did not start")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if m.screen != screenChat {
		t.Fatalf("screen after running cancel = %d, want screenChat", m.screen)
	}
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should clear immediately after local cancellation")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Review interrupted") || !strings.Contains(view, reviewRunnerCancelledMessage) {
		t.Fatalf("View() missing local cancellation state:\n%s", view)
	}

	var finished reviewTimelineRunFinishedMsg
	select {
	case msg := <-done:
		var ok bool
		finished, ok = msg.(reviewTimelineRunFinishedMsg)
		if !ok {
			t.Fatalf("review command msg = %T, want reviewTimelineRunFinishedMsg", msg)
		}
		if !errors.Is(finished.err, context.Canceled) {
			t.Fatalf("review command err = %v, want context canceled", finished.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunReview did not stop after cancelling review")
	}

	updated, _ = m.Update(finished)
	m = updated.(Model)
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should stay clear after canceled command result")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, reviewRunnerCancelledMessage) || strings.Contains(view, "context canceled") {
		t.Fatalf("View() should keep local cancellation state and ignore command error: %q", view)
	}
}

func TestReviewTimeline_CanceledRunIgnoresQueuedSuccessResult(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("reviewTimelineRun = nil, want active timeline run")
	}
	activeID := m.reviewTimelineRun.id

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should clear immediately after local cancellation")
	}

	queuedSuccess := reviewTimelineRunFinishedMsg{
		id:     activeID,
		report: newTUITestReviewReport(),
	}
	updated, _ = m.Update(queuedSuccess)
	m = updated.(Model)
	view := stripANSI(m.View())
	if strings.Contains(view, "Review result") {
		t.Fatalf("canceled timeline result should not append report:\n%s", view)
	}
	if !strings.Contains(view, reviewRunnerCancelledMessage) {
		t.Fatalf("View() should keep local cancellation state:\n%s", view)
	}
}

func TestReviewTimeline_StaleFinishedMessageIsIgnored(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		report:    newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
	}
	if m.reviewTimelineRun == nil {
		t.Fatal("reviewTimelineRun = nil, want active timeline run")
	}
	activeID := m.reviewTimelineRun.id
	stale := reviewTimelineRunFinishedMsg{
		id:     newReviewRunID(int(activeID) + 1),
		report: newTUITestReviewReport(),
	}

	updated, _ = m.Update(stale)
	m = updated.(Model)
	if m.reviewTimelineRun == nil || m.reviewTimelineRun.id != activeID {
		t.Fatal("stale timeline result should not clear the active review run")
	}
	if view := stripANSI(m.View()); strings.Contains(view, "Review result") {
		t.Fatalf("stale timeline result should not append report:\n%s", view)
	}
}

func TestReviewTimeline_RunReviewErrorIsRendered(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		err:       errors.New("provider failed"),
	}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen()

	updated, cmd := m.startReviewTimeline(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("startReviewTimeline() cmd = nil, want async review command")
	}

	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if m.reviewTimelineRun != nil {
		t.Fatal("reviewTimelineRun should clear after error result")
	}
	view := stripANSI(m.View())
	for _, want := range []string{"review · blocked", "provider failed"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

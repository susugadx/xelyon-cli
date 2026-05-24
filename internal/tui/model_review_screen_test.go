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

func TestReviewCommand_OpensPresetScreen(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/review")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/review should not start chat, got cmd %v", cmd)
	}
	if m.screen != screenReview {
		t.Fatalf("screen = %d, want screenReview(%d)", m.screen, screenReview)
	}
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen is nil")
	}
	if m.reviewScreen.mode != reviewScreenPreset {
		t.Fatalf("review mode = %d, want preset", m.reviewScreen.mode)
	}
}

func TestReviewScreen_CurrentChangesPresetCreatesRequest(t *testing.T) {
	m := newReviewTestModel()

	m = sendReviewKey(m, "enter")

	assertReviewRequest(t, m, review.TargetCurrentChanges, "")
	if m.reviewScreen.mode != reviewScreenSubmitted {
		t.Fatalf("review mode = %d, want submitted", m.reviewScreen.mode)
	}
	if view := m.View(); !strings.Contains(view, reviewRunnerNotImplementedMessage) {
		t.Fatalf("View() missing not implemented message: %q", view)
	}
}

func TestReviewScreen_CustomFocusCopyClarifiesScope(t *testing.T) {
	m := newReviewTestModel()

	presetView := stripANSI(m.View())
	for _, want := range []string{"Review current changes", "Review current changes with custom focus"} {
		if !strings.Contains(presetView, want) {
			t.Fatalf("preset view missing %q:\n%s", want, presetView)
		}
	}
	if strings.Contains(presetView, "Custom review instructions") {
		t.Fatalf("preset view should not use old custom instructions label:\n%s", presetView)
	}

	m = sendReviewKey(m, "down")
	m = sendReviewKey(m, "enter")
	customView := stripANSI(m.View())
	for _, want := range []string{
		"Review current changes with custom focus",
		"Reviews all current changes.",
		"Custom focus adjusts priorities; it does not narrow files or diff scope.",
		"It is not a single-finding recheck mode.",
	} {
		if !strings.Contains(customView, want) {
			t.Fatalf("custom focus view missing %q:\n%s", want, customView)
		}
	}
}

func TestReviewScreen_CustomFocusInputVisibleOnShortTerminal(t *testing.T) {
	m := newReviewTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)

	m = sendReviewKey(m, "down")
	m = sendReviewKey(m, "enter")

	view := stripANSI(m.View())
	for _, want := range []string{
		"Review current changes with custom focus",
		"Add custom focus...",
		"Reviews all current changes.",
		"It is not a single-finding recheck mode.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("short custom focus view missing %q:\n%s", want, view)
		}
	}
}

func TestReviewScreen_CustomInstructionsCreatesRequest(t *testing.T) {
	m := newReviewTestModel()

	m = sendReviewKey(m, "down")
	m = sendReviewKey(m, "enter")
	if m.reviewScreen.mode != reviewScreenCustom {
		t.Fatalf("review mode = %d, want custom", m.reviewScreen.mode)
	}

	m = sendReviewText(m, "focus on regressions")
	m = sendReviewKey(m, "enter")

	assertReviewRequest(t, m, review.TargetCurrentChanges, "focus on regressions")
	if m.reviewScreen.mode != reviewScreenSubmitted {
		t.Fatalf("review mode = %d, want submitted", m.reviewScreen.mode)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Custom focus: focus on regressions") {
		t.Fatalf("submitted view missing custom focus label:\n%s", view)
	}
	if strings.Contains(view, "Custom instructions:") {
		t.Fatalf("submitted view should not use old custom instructions label:\n%s", view)
	}
}

func TestReviewScreen_EscBackTargets(t *testing.T) {
	t.Run("preset returns to chat", func(t *testing.T) {
		m := newReviewTestModel()
		m = sendReviewKey(m, "esc")
		if m.screen != screenChat {
			t.Fatalf("screen = %d, want screenChat", m.screen)
		}
		if m.reviewScreen != nil {
			t.Fatal("reviewScreen should be nil after closing")
		}
	})

	t.Run("custom returns to preset", func(t *testing.T) {
		m := newReviewTestModel()
		m = sendReviewKey(m, "down")
		m = sendReviewKey(m, "enter")
		m = sendReviewKey(m, "esc")
		if m.screen != screenReview {
			t.Fatalf("screen = %d, want screenReview", m.screen)
		}
		if m.reviewScreen.mode != reviewScreenPreset {
			t.Fatalf("review mode = %d, want preset", m.reviewScreen.mode)
		}
	})
}

func TestReviewScreen_CloseAfterResize_RebuildsChatFooter(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/review")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = updated.(Model)
	if m.screen != screenReview {
		t.Fatalf("screen after resize = %d, want screenReview", m.screen)
	}

	m = sendReviewKey(m, "esc")
	if m.screen != screenChat {
		t.Fatalf("screen after close = %d, want screenChat", m.screen)
	}
	if m.chromeDirty {
		t.Fatal("chromeDirty should be false because closeReviewScreen rebuilds chrome immediately")
	}
	verifyViewLines(t, m, "review close after resize")
}

func TestReviewScreen_NilReviewAgentShowsNotImplemented(t *testing.T) {
	m := newReviewTestModel()

	updated, cmd := m.handleReviewRequest(review.NewCurrentChangesRequest(""))
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("handleReviewRequest() cmd = %v, want nil", cmd)
	}
	if m.reviewScreen.message != reviewRunnerNotImplementedMessage {
		t.Fatalf("review message = %q, want %q", m.reviewScreen.message, reviewRunnerNotImplementedMessage)
	}
	if view := m.View(); !strings.Contains(view, reviewRunnerNotImplementedMessage) {
		t.Fatalf("View() missing not implemented message: %q", view)
	}
}

func TestReviewScreen_BlocksRunReviewWhileAgentIsProcessing(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{
			statusLine: "processing",
			processing: true,
		},
		report: newTUITestReviewReport(),
	}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen(1)

	updated, cmd := m.handleReviewRequest(review.NewCurrentChangesRequest("focus on regressions"))
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("handleReviewRequest() cmd = %v, want nil while agent is processing", cmd)
	}
	if agent.reviewCalls != 0 {
		t.Fatalf("RunReview calls = %d, want 0 while agent is processing", agent.reviewCalls)
	}
	if m.reviewScreen.activeRun != nil {
		t.Fatal("active review run should not be created while agent is processing")
	}
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("review run state = %d, want failed busy state", m.reviewScreen.runState)
	}
	if m.reviewScreen.message != reviewRunnerBusyMessage {
		t.Fatalf("review message = %q, want %q", m.reviewScreen.message, reviewRunnerBusyMessage)
	}
	if view := m.View(); !strings.Contains(view, reviewRunnerBusyMessage) {
		t.Fatalf("View() missing busy message: %q", view)
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

func TestReviewScreen_RunReviewSuccessIsAsyncAndRendered(t *testing.T) {
	agent := &reviewCapableStubAgent{stubAgent: stubAgent{statusLine: "ready"}}
	agent.report = newTUITestReviewReport()
	m := newModelWithViewport(agent)
	if m.reviewAgent == nil {
		t.Fatal("reviewAgent = nil, want optional ReviewAgent to be captured")
	}

	m.screen = screenReview
	m.reviewScreen = newReviewScreen(1)
	req := review.NewCurrentChangesRequest("")
	updated, cmd := m.handleReviewRequest(req)
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("handleReviewRequest() cmd = nil, want async review command")
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
	if m.reviewScreen.runState != reviewRunSucceeded {
		t.Fatalf("review run state = %d, want succeeded", m.reviewScreen.runState)
	}
	view := m.View()
	for _, want := range []string{"has_findings", "Verification: verified", "Group: request state", "Finding: stale result is ignored"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestReviewScreen_EscWhileRunningCancelsReviewBeforeClose(t *testing.T) {
	agent := newCancellableReviewAgent()
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen(1)

	updated, cmd := m.handleReviewRequest(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("handleReviewRequest() cmd = nil, want async review command")
	}
	if m.reviewScreen.activeRun == nil {
		t.Fatal("review screen should hold active run while review is running")
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

	m = sendReviewKey(m, "esc")
	if m.screen != screenReview {
		t.Fatalf("screen after running cancel = %d, want screenReview", m.screen)
	}
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen should remain visible while cancellation is observed")
	}
	if m.reviewScreen.activeRun != nil {
		t.Fatal("active run should be cleared after requesting cancellation")
	}
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("review run state = %d, want failed after requesting cancellation", m.reviewScreen.runState)
	}
	if m.reviewScreen.message != reviewRunnerCancelledMessage {
		t.Fatalf("review message = %q, want %q", m.reviewScreen.message, reviewRunnerCancelledMessage)
	}

	var finished reviewRunFinishedMsg
	select {
	case msg := <-done:
		var ok bool
		finished, ok = msg.(reviewRunFinishedMsg)
		if !ok {
			t.Fatalf("review command msg = %T, want reviewRunFinishedMsg", msg)
		}
		if !errors.Is(finished.err, context.Canceled) {
			t.Fatalf("review command err = %v, want context canceled", finished.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunReview did not stop after cancelling review")
	}

	updated, _ = m.Update(finished)
	m = updated.(Model)
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen should still show cancellation result")
	}
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("review run state = %d, want failed after canceled command result", m.reviewScreen.runState)
	}
	if view := m.View(); !strings.Contains(view, reviewRunnerCancelledMessage) || strings.Contains(view, "context canceled") {
		t.Fatalf("View() should keep local cancellation state and ignore command error: %q", view)
	}

	m = sendReviewKey(m, "esc")
	if m.screen != screenChat {
		t.Fatalf("screen after completed cancel close = %d, want screenChat", m.screen)
	}
}

func TestReviewScreen_CancelledRunIgnoresQueuedSuccessResult(t *testing.T) {
	m := newReviewTestModel()
	runCtx := m.reviewScreen.startReview(review.NewCurrentChangesRequest(""))
	stale := reviewRunFinishedMsg{
		id:     runCtx.id,
		report: newTUITestReviewReport(),
	}

	m = sendReviewKey(m, "esc")
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen should remain visible after cancellation")
	}
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("review run state = %d, want failed after cancellation", m.reviewScreen.runState)
	}

	updated, _ := m.Update(stale)
	m = updated.(Model)
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("stale success changed run state to %d", m.reviewScreen.runState)
	}
	if m.reviewScreen.report != nil {
		t.Fatal("stale success should not attach report after cancellation")
	}
	if m.reviewScreen.message != reviewRunnerCancelledMessage {
		t.Fatalf("review message = %q, want %q", m.reviewScreen.message, reviewRunnerCancelledMessage)
	}
}

func TestReviewScreen_SubmittedReportCanScrollToHiddenFindings(t *testing.T) {
	m := newReviewTestModel()
	m.width = 80
	m.height = 8
	req := review.NewCurrentChangesRequest("")
	m.reviewScreen.startReview(req)
	report := newLongTUITestReviewReport(12)
	m.reviewScreen.completeReview(report)

	initialView := stripANSI(m.View())
	if strings.Contains(initialView, "hidden finding 11") {
		t.Fatalf("initial review view unexpectedly contains tail finding:\n%s", initialView)
	}
	if !strings.Contains(initialView, "PgUp/PgDn:page") {
		t.Fatalf("overflowing review view should advertise paging controls:\n%s", initialView)
	}

	for i := 0; i < 8; i++ {
		m = sendReviewKey(m, "pgdown")
	}

	scrolledView := stripANSI(m.View())
	if !strings.Contains(scrolledView, "hidden finding 11") {
		t.Fatalf("paged review view missing tail finding:\n%s", scrolledView)
	}
	if !strings.Contains(scrolledView, "Generated at: 2026-01-01T00:00:00Z") {
		t.Fatalf("paged review view missing report tail:\n%s", scrolledView)
	}

	m = sendReviewKey(m, "home")
	if m.reviewScreen.bodyViewport.yOffset != 0 {
		t.Fatalf("home should reset review body offset, got %d", m.reviewScreen.bodyViewport.yOffset)
	}
}

func TestReviewScreen_NewSubmittedResultResetsBodyScroll(t *testing.T) {
	m := newReviewTestModel()
	m.width = 80
	m.height = 8
	m.reviewScreen.startReview(review.NewCurrentChangesRequest(""))
	m.reviewScreen.completeReview(newLongTUITestReviewReport(12))
	m = sendReviewKey(m, "end")
	if m.reviewScreen.bodyViewport.yOffset == 0 {
		t.Fatal("expected end key to move review body offset")
	}

	m.reviewScreen.completeReview(newLongTUITestReviewReport(2))
	if m.reviewScreen.bodyViewport.yOffset != 0 {
		t.Fatalf("new review result should reset body offset, got %d", m.reviewScreen.bodyViewport.yOffset)
	}
}

func TestReviewScreen_RunReviewErrorIsRendered(t *testing.T) {
	agent := &reviewCapableStubAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		err:       errors.New("provider failed"),
	}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen(1)

	updated, cmd := m.handleReviewRequest(review.NewCurrentChangesRequest(""))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("handleReviewRequest() cmd = nil, want async review command")
	}

	updated, _ = m.Update(cmd())
	m = updated.(Model)
	if m.reviewScreen.runState != reviewRunFailed {
		t.Fatalf("review run state = %d, want failed", m.reviewScreen.runState)
	}
	if view := m.View(); !strings.Contains(view, "provider failed") {
		t.Fatalf("View() missing error message: %q", view)
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

	if m.screen != screenReview {
		t.Fatalf("screen = %d, want screenReview(%d)", m.screen, screenReview)
	}
	assertReviewRequest(t, m, review.TargetCurrentChanges, "focus on regressions")
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

	assertReviewRequest(t, m, review.TargetCurrentChanges, `"focus on regressions"`)
	if cmd == nil {
		t.Fatal("/review with quoted instructions cmd = nil, want async review command")
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

	assertReviewRequest(t, m, review.TargetCurrentChanges, `investigate "quoted`)
	if cmd == nil {
		t.Fatal("/review with unterminated quote instructions cmd = nil, want async review command")
	}

	for _, msg := range runReviewCommandForTest(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	if got := agent.lastRequest.CustomInstructions; got != `investigate "quoted` {
		t.Fatalf("RunReview CustomInstructions = %q, want raw instructions", got)
	}
}

func TestReviewScreen_StaleFinishedMessageIsIgnored(t *testing.T) {
	m := newReviewTestModel()
	m.reviewScreen.startReview(review.NewCurrentChangesRequest(""))
	stale := reviewRunFinishedMsg{
		id:     newReviewRunID(m.reviewScreen.screenID, m.reviewScreen.runSeq),
		report: newTUITestReviewReport(),
	}

	updated, _ := m.closeReviewScreen()
	m = updated.(Model)
	updated, _ = m.openReviewScreen()
	m = updated.(Model)

	updated, _ = m.Update(stale)
	m = updated.(Model)
	if m.reviewScreen.runState != reviewRunIdle {
		t.Fatalf("stale review result changed run state to %d", m.reviewScreen.runState)
	}
	if m.reviewScreen.report != nil {
		t.Fatal("stale review result should not attach report to reopened screen")
	}
}

func newReviewTestModel() Model {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreenSeq = 1
	m.reviewScreen = newReviewScreen(m.reviewScreenSeq)
	m.reviewScreen.customInput.Width = m.width - 4
	m.rebuildChrome()
	return m
}

type reviewCapableStubAgent struct {
	stubAgent
	reviewCalls int
	lastRequest review.ReviewRequest
	report      review.ReviewReport
	err         error
}

type cancellableReviewAgent struct {
	stubAgent
	started chan struct{}
}

func newCancellableReviewAgent() *cancellableReviewAgent {
	return &cancellableReviewAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		started:   make(chan struct{}),
	}
}

func (s *cancellableReviewAgent) RunReview(ctx context.Context, _ review.ReviewRequest) (review.ReviewReport, error) {
	close(s.started)
	<-ctx.Done()
	return review.ReviewReport{}, ctx.Err()
}

func (s *reviewCapableStubAgent) RunReview(_ context.Context, req review.ReviewRequest) (review.ReviewReport, error) {
	s.reviewCalls++
	s.lastRequest = req
	return s.report, s.err
}

func runReviewCommandForTest(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		msgs := make([]tea.Msg, 0, len(batch))
		for _, batchCmd := range batch {
			if batchCmd == nil {
				continue
			}
			if batchMsg := batchCmd(); batchMsg != nil {
				msgs = append(msgs, batchMsg)
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func sendReviewKey(m Model, s string) Model {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "pgup":
		msg = tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		msg = tea.KeyMsg{Type: tea.KeyPgDown}
	case "home":
		msg = tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		msg = tea.KeyMsg{Type: tea.KeyEnd}
	default:
		if len(s) == 1 {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func sendReviewText(m Model, text string) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return updated.(Model)
}

func assertReviewRequest(t *testing.T, m Model, target review.TargetKind, custom string) {
	t.Helper()
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen is nil")
	}
	if m.reviewScreen.request == nil {
		t.Fatal("review request is nil")
	}
	if m.reviewScreen.request.TargetKind != target {
		t.Fatalf("TargetKind = %q, want %q", m.reviewScreen.request.TargetKind, target)
	}
	if m.reviewScreen.request.CustomInstructions != custom {
		t.Fatalf("CustomInstructions = %q, want %q", m.reviewScreen.request.CustomInstructions, custom)
	}
}

package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

type reviewRunID struct {
	screenID int
	runSeq   int
}

func newReviewRunID(screenID int, runSeq int) reviewRunID {
	return reviewRunID{
		screenID: screenID,
		runSeq:   runSeq,
	}
}

func (id reviewRunID) matchesActiveRun(rs *reviewScreen) bool {
	return rs != nil && rs.activeRun != nil && id == rs.activeRun.id
}

type reviewRunContext struct {
	id     reviewRunID
	ctx    context.Context
	cancel context.CancelFunc
}

func newReviewRunContext(id reviewRunID) *reviewRunContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &reviewRunContext{
		id:     id,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *reviewRunContext) cancelRun() {
	if r == nil || r.cancel == nil {
		return
	}
	r.cancel()
}

func (rs *reviewScreen) cancelActiveReviewRun() {
	if rs == nil || rs.activeRun == nil {
		return
	}
	rs.activeRun.cancelRun()
	rs.activeRun = nil
}

func (rs *reviewScreen) clearActiveReviewRun() {
	if rs != nil {
		rs.activeRun = nil
	}
}

func (rs *reviewScreen) startActiveReviewRun() *reviewRunContext {
	rs.cancelActiveReviewRun()
	id := newReviewRunID(rs.screenID, rs.runSeq)
	rs.activeRun = newReviewRunContext(id)
	return rs.activeRun
}

type reviewRunInvocation struct {
	id      reviewRunID
	ctx     context.Context
	agent   ReviewAgent
	request review.ReviewRequest
}

func newReviewRunInvocation(runCtx *reviewRunContext, agent ReviewAgent, req review.ReviewRequest) reviewRunInvocation {
	reqCopy := req
	return reviewRunInvocation{
		id:      runCtx.id,
		ctx:     runCtx.ctx,
		agent:   agent,
		request: reqCopy,
	}
}

func (r reviewRunInvocation) command() tea.Cmd {
	return func() tea.Msg {
		report, err := r.agent.RunReview(r.ctx, r.request)
		return reviewRunFinishedMsg{
			id:     r.id,
			report: report,
			err:    err,
		}
	}
}

type reviewRunFinishedMsg struct {
	id     reviewRunID
	report review.ReviewReport
	err    error
}

func (msg reviewRunFinishedMsg) appliesTo(rs *reviewScreen) bool {
	return msg.id.matchesActiveRun(rs)
}

func (msg reviewRunFinishedMsg) applyTo(rs *reviewScreen) {
	rs.clearActiveReviewRun()
	if msg.err != nil {
		rs.failReview(msg.err)
		return
	}
	rs.completeReview(msg.report)
}

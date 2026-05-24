package tui

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

type reviewRunID int

func newReviewRunID(seq int) reviewRunID {
	return reviewRunID(seq)
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

func (m *Model) interruptReviewTimelineRun() bool {
	if m.reviewTimelineRun == nil {
		return false
	}

	m.reviewTimelineRun.cancelRun()
	m.reviewTimelineRun = nil
	m.conversation.Cancel()
	m.resetStreamingState()
	m.refreshStatusLine()
	err := errors.New(reviewRunnerCancelledMessage)
	m.finishAgentActivity(err, AgentErrorKindFromError(err, AgentErrorUnknown))
	return true
}

type reviewTimelineRunInvocation struct {
	id      reviewRunID
	ctx     context.Context
	agent   ReviewAgent
	request review.ReviewRequest
}

func newReviewTimelineRunInvocation(runCtx *reviewRunContext, agent ReviewAgent, req review.ReviewRequest) reviewTimelineRunInvocation {
	reqCopy := req
	return reviewTimelineRunInvocation{
		id:      runCtx.id,
		ctx:     runCtx.ctx,
		agent:   agent,
		request: reqCopy,
	}
}

func (r reviewTimelineRunInvocation) command() tea.Cmd {
	return func() tea.Msg {
		report, err := r.agent.RunReview(r.ctx, r.request)
		return reviewTimelineRunFinishedMsg{
			id:     r.id,
			report: report,
			err:    err,
		}
	}
}

type reviewTimelineRunFinishedMsg struct {
	id     reviewRunID
	report review.ReviewReport
	err    error
}

func (msg reviewTimelineRunFinishedMsg) appliesTo(m Model) bool {
	return m.reviewTimelineRun != nil && msg.id == m.reviewTimelineRun.id
}

func (m Model) handleReviewTimelineFinishedMsg(msg reviewTimelineRunFinishedMsg) (tea.Model, tea.Cmd) {
	if !msg.appliesTo(m) {
		return m, nil
	}
	m.reviewTimelineRun = nil
	m.resetStreamingState()
	m.refreshStatusLine()
	if msg.err != nil {
		err := msg.err
		if errors.Is(err, context.Canceled) {
			err = errors.New(reviewRunnerCancelledMessage)
		}
		m.finishAgentActivity(err, AgentErrorKindFromError(err, AgentErrorUnknown))
		return m, nil
	}
	m.finishAgentActivity(nil, AgentErrorUnknown)
	return m, m.appendMessage(ChatMessage{
		Role:    "assistant",
		Content: reviewReportTimelineMessage(msg.report),
	})
}

func reviewTimelineUserDisplay(req review.ReviewRequest) string {
	base := "/review current changes"
	if strings.TrimSpace(req.CustomInstructions) == "" {
		return base
	}
	return base + "\nfocus: " + termtext.SanitizeSingleLineANSI(req.CustomInstructions)
}

func reviewActivityOptions(_ review.ReviewRequest) agentActivityOptions {
	return agentActivityOptions{
		title:       "review",
		workingText: "running current changes review",
		doneText:    "completed current changes review",
		hideStatus:  true,
	}
}

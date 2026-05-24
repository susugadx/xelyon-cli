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
	ctx, cancel := context.WithCancel(contextWithReviewRunID(context.Background(), id))
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
		result, err := r.agent.RunReview(r.ctx, r.request)
		return reviewTimelineRunFinishedMsg{
			id:     r.id,
			result: result,
			err:    err,
		}
	}
}

type reviewTimelineRunFinishedMsg struct {
	id     reviewRunID
	result ReviewRunResult
	err    error
}

func (msg reviewTimelineRunFinishedMsg) appliesTo(m Model) bool {
	return m.reviewTimelineRun != nil && msg.id == m.reviewTimelineRun.id
}

func (msg ReviewProgressMsg) appliesTo(m Model) bool {
	return m.reviewTimelineRun != nil && reviewRunID(msg.RunID) == m.reviewTimelineRun.id
}

func (m Model) handleReviewProgressMsg(msg ReviewProgressMsg) (tea.Model, tea.Cmd) {
	if !msg.appliesTo(m) {
		return m, nil
	}
	return m, m.handleAppendToolResultMsg(AppendToolResultMsg{Tool: msg.Tool})
}

func (m Model) handleReviewTimelineFinishedMsg(msg reviewTimelineRunFinishedMsg) (tea.Model, tea.Cmd) {
	if !msg.appliesTo(m) {
		return m, nil
	}
	m.reviewTimelineRun = nil
	m.resetStreamingState()
	m.refreshStatusLine()
	m.showReviewRunUsageStatus(msg.result.Usage)
	if msg.err != nil {
		err := msg.err
		if errors.Is(err, context.Canceled) {
			err = errors.New(reviewRunnerCancelledMessage)
		}
		m.finishAgentActivity(err, AgentErrorKindFromError(err, AgentErrorUnknown))
		m.rebuildChromeIfDirty()
		return m, nil
	}
	if doneText := reviewDoneText(msg.result.Usage); doneText != "" {
		m.agentActivity.doneText = doneText
	}
	m.finishAgentActivity(nil, AgentErrorUnknown)
	cmd := m.appendMessage(ChatMessage{
		Role:    "assistant",
		Content: reviewRunTimelineMessage(msg.result),
	})
	m.rebuildChromeIfDirty()
	return m, cmd
}

func (m *Model) showReviewRunUsageStatus(summary ReviewRunUsageSummary) {
	if status := summary.statusText(); status != "" {
		m.setTransientStatus(status)
	}
}

func reviewDoneText(summary ReviewRunUsageSummary) string {
	base := "completed current changes review"
	if usage := summary.inlineText(); usage != "" {
		return base + " · " + usage
	}
	return base
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

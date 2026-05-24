package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

// openReviewScreen は review preset screen を開く。
func (m Model) openReviewScreen() (tea.Model, tea.Cmd) {
	m.activateModalScreen(screenReview)
	m.reviewScreen = newReviewScreen()
	m.reviewScreen.customInput.Width = max(0, m.width-4)
	return m, nil
}

// openReviewScreenAndRun は /review <instructions> の即時実行用に timeline review を開始する。
func (m Model) openReviewScreenAndRun(customInstructions string) (tea.Model, tea.Cmd) {
	req := review.NewCurrentChangesRequest(customInstructions)
	return m.startReviewTimeline(req)
}

// closeReviewScreen は review screen を閉じて chat に戻る。
func (m Model) closeReviewScreen() (tea.Model, tea.Cmd) {
	m.reviewScreen = nil
	m.deactivateModalScreen(true)
	return m, nil
}

// updateReviewScreen は screenReview 中のメッセージ処理。
func (m Model) updateReviewScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	rs := m.reviewScreen
	if rs == nil {
		return m.closeReviewScreen()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyChatWindowSize(msg.Width, msg.Height)
		rs.customInput.Width = max(0, msg.Width-4)
		return m, nil

	case tea.MouseMsg:
		m.screen = screenChat
		updated, cmd := m.Update(msg)
		m = updated.(Model)
		m.screen = screenReview
		return m, cmd

	case tea.KeyMsg:
		action, req, cmd := rs.handleKey(msg)
		switch action {
		case reviewCommandDelegateCtrlC:
			return m.handleCtrlC()
		case reviewCommandClose:
			return m.closeReviewScreen()
		case reviewCommandSubmit:
			if req == nil {
				return m, cmd
			}
			updated, reviewCmd := m.startReviewTimeline(*req)
			return updated, tea.Batch(cmd, reviewCmd)
		default:
			return m, cmd
		}

	default:
		return m.forwardMessageToChatFromModal(msg, screenReview)
	}
}

func (m Model) startReviewTimeline(req review.ReviewRequest) (tea.Model, tea.Cmd) {
	if m.rejectReviewTimelineWhileBusy() {
		return m, nil
	}
	m.prepareReviewTimelineHandoff()

	m.appendUserMessage(reviewTimelineUserDisplay(req))
	m.beginAgentActivityWithOptions(reviewActivityOptions(req))
	if m.reviewAgent == nil {
		err := errors.New(reviewRunnerNotImplementedMessage)
		m.finishAgentActivity(err, AgentErrorValidation)
		return m, nil
	}

	m.reviewTimelineSeq++
	runCtx := newReviewRunContext(newReviewRunID(m.reviewTimelineSeq))
	m.reviewTimelineRun = runCtx
	agent := m.reviewAgent
	return m, newReviewTimelineRunInvocation(runCtx, agent, req).command()
}

func (m *Model) rejectReviewTimelineWhileBusy() bool {
	if !m.rejectAgentTurnWhileBusy() {
		return false
	}
	m.showReviewBusyNotice()
	return true
}

func (m *Model) showReviewBusyNotice() {
	if m.screen != screenReview || m.reviewScreen == nil {
		return
	}
	m.reviewScreen.setNotice(agentTurnBusyStatus)
	if m.reviewScreen.mode == reviewScreenCustom {
		m.reviewScreen.customInput.Focus()
	}
	m.chromeDirty = true
}

func (m *Model) prepareReviewTimelineHandoff() {
	m.clearComposerDraft()
	if m.reviewScreen != nil {
		m.reviewScreen = nil
	}
	if m.screen != screenChat {
		m.deactivateModalScreen(false)
	}
}

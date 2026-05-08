package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/review"
)

// openReviewScreen は review preset screen を開く。
func (m Model) openReviewScreen() (tea.Model, tea.Cmd) {
	m.activateModalScreen(screenReview)
	m.reviewScreenSeq++
	m.reviewScreen = newReviewScreen(m.reviewScreenSeq)
	m.reviewScreen.customInput.Width = max(0, m.width-4)
	return m, nil
}

// openReviewScreenAndRun は /review <instructions> の即時実行用に画面を開いて request を走らせる。
func (m Model) openReviewScreenAndRun(customInstructions string) (tea.Model, tea.Cmd) {
	updated, cmd := m.openReviewScreen()
	m = updated.(Model)
	req := review.NewCurrentChangesRequest(customInstructions)
	updated, reviewCmd := m.handleReviewRequest(req)
	return updated, tea.Batch(cmd, reviewCmd)
}

// closeReviewScreen は review screen を閉じて chat に戻る。
func (m Model) closeReviewScreen() (tea.Model, tea.Cmd) {
	if m.reviewScreen != nil {
		m.reviewScreen.cancelActiveReviewRun()
	}
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
	case reviewRunFinishedMsg:
		if !msg.appliesTo(rs) {
			return m, nil
		}
		msg.applyTo(rs)
		m.chromeDirty = true
		return m, nil

	case tea.WindowSizeMsg:
		m.applyChatWindowSize(msg.Width, msg.Height)
		rs.customInput.Width = max(0, msg.Width-4)
		rs.bodyViewport.clamp(m.reviewBodyScrollBounds())
		return m, nil

	case tea.MouseMsg:
		if rs.handleMouse(msg, m.reviewBodyScrollBounds()) {
			return m, nil
		}
		m.screen = screenChat
		updated, cmd := m.Update(msg)
		m = updated.(Model)
		m.screen = screenReview
		return m, cmd

	case tea.KeyMsg:
		action, cmd := rs.handleKey(msg, m.reviewBodyScrollBounds())
		switch action {
		case reviewCommandDelegateCtrlC:
			return m.handleCtrlC()
		case reviewCommandClose:
			return m.closeReviewScreen()
		case reviewCommandSubmit:
			if rs.request == nil {
				return m, cmd
			}
			updated, reviewCmd := m.handleReviewRequest(*rs.request)
			return updated, tea.Batch(cmd, reviewCmd)
		default:
			return m, cmd
		}

	default:
		return m.forwardMessageToChatFromModal(msg, screenReview)
	}
}

func (m Model) reviewBodyScrollBounds() reviewBodyScrollBounds {
	return newReviewBodyScrollBounds(len(m.reviewBodyLines()), m.reviewBodyHeight())
}

// handleReviewRequest は生成済み ReviewRequest の runner handoff 境界。
func (m Model) handleReviewRequest(req review.ReviewRequest) (tea.Model, tea.Cmd) {
	if m.reviewScreen == nil {
		return m, nil
	}
	if m.reviewAgent == nil {
		m.reviewScreen.markReviewNotImplemented(req)
		return m, nil
	}
	if m.conversation != nil && m.conversation.IsProcessing() {
		m.reviewScreen.markReviewBlocked(req, reviewRunnerBusyMessage)
		return m, nil
	}

	runCtx := m.reviewScreen.startReview(req)
	agent := m.reviewAgent
	return m, newReviewRunInvocation(runCtx, agent, req).command()
}

package tui

import (
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

// closeReviewScreen は review screen を閉じて chat に戻る。
func (m Model) closeReviewScreen() (tea.Model, tea.Cmd) {
	m.reviewScreen = nil
	m.deactivateModalScreen(true)
	return m, nil
}

// updateReviewScreen は screenReview 中のメッセージ処理。
// review runner は未実装のため、現時点では ReviewRequest を作るところまでを担当する。
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

	case tea.KeyMsg:
		action, cmd := rs.handleKey(msg)
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

// handleReviewRequest は生成済み ReviewRequest の runner handoff 境界。
// runner 未実装の間は request を保持し、未実装メッセージだけを表示する。
func (m Model) handleReviewRequest(req review.ReviewRequest) (tea.Model, tea.Cmd) {
	if m.reviewScreen != nil {
		reqCopy := req
		m.reviewScreen.request = &reqCopy
		m.reviewScreen.message = reviewRunnerNotImplementedMessage
	}
	return m, nil
}

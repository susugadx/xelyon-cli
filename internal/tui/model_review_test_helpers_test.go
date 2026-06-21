package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/reviewscreen"
)

func newReviewTestModel() Model {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = reviewscreen.New(m.width)
	m.rebuildChrome()
	return m
}

func newReviewCapableTestModel(agent *reviewCapableStubAgent) Model {
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = reviewscreen.New(m.width)
	m.rebuildChrome()
	return m
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
	updated, _ := sendReviewKeyWithCmd(m, s)
	return updated
}

func sendReviewKeyWithCmd(m Model, s string) (Model, tea.Cmd) {
	msg := reviewKeyMsg(s)
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func reviewKeyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		if len(s) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
	}
	return tea.KeyMsg{}
}

func applyReviewCommandMessages(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for _, msg := range runReviewCommandForTest(t, cmd) {
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}
	return m
}

func sendReviewText(m Model, text string) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return updated.(Model)
}

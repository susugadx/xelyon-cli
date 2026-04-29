package tui

import (
	"strings"
	"testing"

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

func TestReviewScreen_UncommittedPresetCreatesRequest(t *testing.T) {
	m := newReviewTestModel()

	m = sendReviewKey(m, "enter")

	assertReviewRequest(t, m, review.TargetUncommitted, "")
	if m.reviewScreen.mode != reviewScreenSubmitted {
		t.Fatalf("review mode = %d, want submitted", m.reviewScreen.mode)
	}
	if view := m.View(); !strings.Contains(view, reviewRunnerNotImplementedMessage) {
		t.Fatalf("View() missing not implemented message: %q", view)
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

	assertReviewRequest(t, m, review.TargetUncommitted, "focus on regressions")
	if m.reviewScreen.mode != reviewScreenSubmitted {
		t.Fatalf("review mode = %d, want submitted", m.reviewScreen.mode)
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

func newReviewTestModel() Model {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.screen = screenReview
	m.reviewScreen = newReviewScreen()
	m.reviewScreen.customInput.Width = m.width - 4
	m.rebuildChrome()
	return m
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

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

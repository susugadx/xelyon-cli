package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashSuggestions_ShowOnSlashAndRenderDescription(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)

	if !m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should be visible after typing /")
	}
	if got := len(m.visibleSlashSuggestionRows()); got == 0 {
		t.Fatal("visible slash suggestion rows should not be empty")
	}
	if m.vp.height != m.height-m.footerHeight() {
		t.Fatalf("viewport height = %d, want %d after suggestions", m.vp.height, m.height-m.footerHeight())
	}

	rendered := stripANSI(m.chromeCache)
	for _, fragment := range []string{"/review", "Review current changes and find issues"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("chromeCache missing %q:\n%s", fragment, rendered)
		}
	}
}

func TestSlashSuggestions_FilterOnPrefix(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m = sendComposerRunes(m, "/r")

	if !m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should be visible after typing /r")
	}
	if got := len(m.slashSuggestions.suggestions); got != 1 {
		t.Fatalf("suggestions len = %d, want 1", got)
	}
	if got := m.slashSuggestions.suggestions[0].InsertText; got != "/review" {
		t.Fatalf("suggestion = %q, want /review", got)
	}
}

func TestSlashSuggestions_EnterExecutesSelectedCommand(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/r")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("selected /review should not start chat, got cmd %v", cmd)
	}
	if m.screen != screenReview {
		t.Fatalf("screen = %d, want screenReview", m.screen)
	}
	if m.reviewScreen == nil {
		t.Fatal("reviewScreen is nil")
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after command execution")
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != "/review" {
		t.Fatalf("last message should be /review, got %#v", m.messages)
	}
}

func TestSlashSuggestions_TabCompletesSelectedCommand(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/r")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Tab completion should not execute command, got %v", cmd)
	}
	if got := m.textInput.Value(); got != "/review" {
		t.Fatalf("textInput after Tab = %q, want /review", got)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after Tab completion")
	}
	if m.screen != screenChat {
		t.Fatalf("screen after Tab = %d, want screenChat", m.screen)
	}
}

func TestSlashSuggestions_ShowThinkingArgumentSuggestions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking ")

	if !m.slashSuggestions.visible() {
		t.Fatal("thinking argument suggestions should be visible")
	}
	rendered := stripANSI(m.chromeCache)
	if !strings.Contains(rendered, "/thinking xhigh (max)") {
		t.Fatalf("chromeCache missing xhigh max suggestion:\n%s", rendered)
	}
}

func TestSlashSuggestions_TabCompletesThinkingArgument(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking x")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Tab completion should not execute command, got %v", cmd)
	}
	if got := m.textInput.Value(); got != "/thinking xhigh" {
		t.Fatalf("textInput after Tab = %q, want /thinking xhigh", got)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after argument completion")
	}
}

func TestSlashSuggestions_EnterExecutesThinkingArgument(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/thinking xhigh": true},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/thinking x")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("selected /thinking xhigh should not start chat, got cmd %v", cmd)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/thinking xhigh" {
		t.Fatalf("handledInputs[0] = %q, want /thinking xhigh", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
}

func TestSlashSuggestions_EnterPreservesThinkingStatusWithTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		dispatched string
	}{
		{name: "canonical", input: "/thinking ", dispatched: "/thinking"},
		{name: "alias", input: "/think ", dispatched: "/think"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &stubAgent{
				statusLine:      "ready",
				handledCommands: map[string]bool{tt.dispatched: true},
			}
			m := newModelWithViewport(agent)
			m = sendComposerRunes(m, tt.input)

			if !m.slashSuggestions.visible() {
				t.Fatal("thinking argument suggestions should be visible before Enter")
			}

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			if cmd != nil {
				t.Fatalf("no-arg thinking status command should not start chat, got cmd %v", cmd)
			}
			if got := len(agent.handledInputs); got != 1 {
				t.Fatalf("handledInputs length = %d, want 1", got)
			}
			if got := agent.handledInputs[0]; got != tt.dispatched {
				t.Fatalf("handledInputs[0] = %q, want %q", got, tt.dispatched)
			}
			if m.textInput.Value() != "" {
				t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
			}
		})
	}
}

func TestSlashSuggestions_EnterExecutesSkillsSubcommand(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/skills doctor": true},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills d")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("selected /skills doctor should not start chat, got cmd %v", cmd)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/skills doctor" {
		t.Fatalf("handledInputs[0] = %q, want /skills doctor", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
}

func TestSlashSuggestions_TabCompletesSkillsSubcommandWithRequiredArg(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills sh")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Tab completion should not execute command, got %v", cmd)
	}
	if got := m.textInput.Value(); got != "/skills show " {
		t.Fatalf("textInput after Tab = %q, want '/skills show '", got)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after subcommand completion")
	}
}

func TestSlashSuggestions_CanMoveSelectionWithDownAndTab(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Tab completion should not execute command, got %v", cmd)
	}
	if got := m.textInput.Value(); got != "/use " {
		t.Fatalf("textInput after Down+Tab = %q, want /use with trailing space", got)
	}
}

func TestSlashSuggestions_CapRowsToSmallWindow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)

	m = sendComposerRunes(m, "/")

	wantRows := m.height - statusBarHeight - inputHeight - minChatViewportHeight
	if got := len(m.visibleSlashSuggestionRows()); got != wantRows {
		t.Fatalf("visible slash suggestion rows = %d, want %d", got, wantRows)
	}
	if m.vp.height != minChatViewportHeight {
		t.Fatalf("viewport height = %d, want %d", m.vp.height, minChatViewportHeight)
	}
	if got := m.vp.height + m.footerHeight(); got != m.height {
		t.Fatalf("vp.height + footerHeight = %d, want %d", got, m.height)
	}
	verifyViewLines(t, m, "small slash suggestions")
}

func TestSlashSuggestions_NoHiddenCompletionWhenNoRowsFit(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: statusBarHeight + inputHeight + minChatViewportHeight})
	m = updated.(Model)
	m = sendComposerRunes(m, "/r")

	if !m.slashSuggestions.visible() {
		t.Fatal("slash suggestion state should keep matches for later resize")
	}
	if got := len(m.visibleSlashSuggestionRows()); got != 0 {
		t.Fatalf("visible slash suggestion rows = %d, want 0", got)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("hidden slash suggestions should not complete, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/r" {
		t.Fatalf("textInput after Tab with hidden suggestions = %q, want /r", got)
	}
}

func TestSlashSuggestions_EscClosesWithoutEnteringNavigation(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Esc should only close suggestions, got cmd %v", cmd)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close on Esc")
	}
	if m.navigationMode {
		t.Fatal("Esc closing suggestions should not enter navigation mode")
	}
	if got := m.textInput.Value(); got != "/" {
		t.Fatalf("textInput after Esc = %q, want /", got)
	}
}

func TestSlashSuggestions_RefreshAfterInlinePaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	m.handleComposerPaste("zz")

	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should refresh after paste and close for unmatched prefix")
	}
	if got := m.textInput.Value(); got != "/zz" {
		t.Fatalf("textInput after paste = %q, want /zz", got)
	}
	if m.vp.height != m.height-m.footerHeight() {
		t.Fatalf("viewport height = %d, want %d after paste", m.vp.height, m.height-m.footerHeight())
	}
}

func TestSlashSuggestions_CloseAfterFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/")

	m.handleComposerPaste("line 1\nline 2\nline 3")

	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after folded paste changes composer mode")
	}
	if !m.hasFoldedPasteBlocks() {
		t.Fatal("paste block should be folded")
	}
	if m.vp.height != m.height-m.footerHeight() {
		t.Fatalf("viewport height = %d, want %d after folded paste", m.vp.height, m.height-m.footerHeight())
	}
}

func sendComposerRunes(m Model, input string) Model {
	for _, r := range input {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	return m
}

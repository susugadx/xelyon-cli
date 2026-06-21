package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tui/slashsuggestions"
)

func TestSlashSuggestions_EnterOnSkillsCommandOpensSubcommandSuggestions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills should open subcommand suggestions, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/skills " {
		t.Fatalf("textInput after Enter = %q, want '/skills '", got)
	}
	if !m.slashSuggestions.Visible() {
		t.Fatal("skills subcommand suggestions should be visible after Enter on selected /skills")
	}
	if !m.slashSuggestions.Snapshot().SelectionActive {
		t.Fatal("Enter-expanded skills suggestions should keep selection active")
	}
	if got := len(m.slashSuggestions.Snapshot().Suggestions); got != 5 {
		t.Fatalf("skills subcommand suggestions len = %d, want 5", got)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_EnterOnSkillsCommandOpensSkillPromptPicker(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "bug-investigation",
				Description: "Investigate known bugs before editing",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills should open skill prompt picker, got cmd %v", cmd)
	}
	suggestion, ok := m.slashSuggestions.SelectedSuggestion()
	if !ok || suggestion.Label != "bug-investigation" || !suggestion.CompleteOnEnter {
		t.Fatalf("selected suggestion = %#v, %v, want skill prompt candidate", suggestion, ok)
	}
	if !m.slashSuggestions.Snapshot().SelectionActive {
		t.Fatal("expanded skill picker should allow Enter to choose the highlighted skill")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on selected skill prompt should paste into composer, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "Use the bug-investigation skill. " {
		t.Fatalf("textInput after selecting skill = %q, want skill prompt reference", got)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := len(agent.chatInputs); got != 0 {
		t.Fatalf("chatInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_EnterOnSkillsCommandExpandsEvenWhenSuggestionRowsAreStale(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/skills": true},
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "bug-investigation",
				Description: "Investigate known bugs before editing",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills")
	m.slashSuggestions = slashsuggestions.State{}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("stale /skills suggestions should expand locally, got cmd %v", cmd)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != "/skills " {
		t.Fatalf("textInput after Enter = %q, want '/skills '", got)
	}
	suggestion, ok := m.slashSuggestions.SelectedSuggestion()
	if !ok || suggestion.Label != "bug-investigation" {
		t.Fatalf("selected suggestion = %#v, %v, want skill prompt candidate", suggestion, ok)
	}
}

func TestSlashSuggestions_EnterOnSkillsShowCompletesRequiredNameArgument(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if suggestion, ok := m.slashSuggestions.SelectedSuggestion(); !ok || suggestion.InsertText != "/skills show" {
		t.Fatalf("selected suggestion = %#v, %v, want /skills show", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills show should complete required name argument, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/skills show " {
		t.Fatalf("textInput after Enter = %q, want '/skills show '", got)
	}
	if m.slashSuggestions.Visible() {
		t.Fatal("slash suggestions should close when no skill name candidates exist")
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_EnterOnTypedSkillsShowCompletesRequiredNameArgument(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "prefix", input: "/skills sh"},
		{name: "subcommand", input: "/skills show"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &stubAgent{statusLine: "ready"}
			m := newModelWithViewport(agent)
			m = sendComposerRunes(m, tt.input)

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			if cmd != nil {
				t.Fatalf("Enter on %q should complete required name argument, got cmd %v", tt.input, cmd)
			}
			if got := m.textInput.Value(); got != "/skills show " {
				t.Fatalf("textInput after Enter = %q, want '/skills show '", got)
			}
			if m.slashSuggestions.Visible() {
				t.Fatal("slash suggestions should close when no skill name candidates exist")
			}
			if got := len(agent.handledInputs); got != 0 {
				t.Fatalf("handledInputs length = %d, want 0", got)
			}
		})
	}
}

func TestSlashSuggestions_ExactSkillsSubcommandWinsOverSkillPromptCandidate(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
	}{
		{name: "doctor", subcommand: "doctor"},
		{name: "overview", subcommand: "overview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := "/skills " + tt.subcommand
			agent := &stubAgent{
				statusLine:      "ready",
				handledCommands: map[string]bool{command: true},
				skillCatalog: agentskills.SkillCatalog{
					Skills: []agentskills.ParsedSkill{{
						Name:        tt.subcommand,
						Description: "Skill with a subcommand name",
					}},
				},
			}
			m := newModelWithViewport(agent)
			m = sendComposerRunes(m, command)

			suggestion, ok := m.slashSuggestions.SelectedSuggestion()
			if !ok || suggestion.InsertText != command || suggestion.CompleteOnEnter {
				t.Fatalf("selected suggestion = %#v, %v, want %s subcommand first", suggestion, ok, command)
			}

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			requireAgentDoneCmd(t, cmd)
			if got := len(agent.handledInputs); got != 1 {
				t.Fatalf("handledInputs length = %d, want 1", got)
			}
			if got := agent.handledInputs[0]; got != command {
				t.Fatalf("handledInputs[0] = %q, want %s", got, command)
			}
			if got := m.textInput.Value(); got != "" {
				t.Fatalf("textInput after command = %q, want empty", got)
			}
		})
	}
}

func TestSlashSuggestions_ExactSkillsShowSubcommandWinsOverSkillPromptCandidate(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "show",
				Description: "Skill with a required-argument subcommand name",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills show")

	suggestion, ok := m.slashSuggestions.SelectedSuggestion()
	if !ok || suggestion.InsertText != "/skills show" || !suggestion.ExpandOnEnter {
		t.Fatalf("selected suggestion = %#v, %v, want /skills show subcommand first", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills show should expand required name argument, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "/skills show " {
		t.Fatalf("textInput after Enter = %q, want '/skills show '", got)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_SkillsListAliasWinsOverSkillPromptCandidate(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/skills list": true},
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "list",
				Description: "Skill with the legacy list alias name",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills list")

	if m.slashSuggestions.Visible() {
		t.Fatalf("exact /skills list alias should not be shadowed by skill suggestions: %#v", m.slashSuggestions.Snapshot().Suggestions)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/skills list" {
		t.Fatalf("handledInputs[0] = %q, want /skills list", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput after command = %q, want empty", got)
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

	requireAgentDoneCmd(t, cmd)
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
	if m.slashSuggestions.Visible() {
		t.Fatal("slash suggestions should close after subcommand completion")
	}
}

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestSlashSuggestions_ShowSkillNameCandidatesWithDescriptions(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{
				{
					Name:        "bug-investigation",
					Description: "Investigate known bugs before editing",
					Source:      agentskills.SourceProject,
					Scripts:     []string{"scripts/repro.sh"},
				},
				{
					Name:        "skill-creator",
					Description: "Create or update Codex skills",
					Source:      agentskills.SourceHome,
					References:  []string{"references/guide.md"},
				},
			},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills show ")

	if !m.slashSuggestions.visible() {
		t.Fatal("skill name suggestions should be visible")
	}
	if got := len(m.slashSuggestions.suggestions); got != 2 {
		t.Fatalf("skill suggestions len = %d, want 2", got)
	}
	suggestion := m.slashSuggestions.suggestions[0]
	if suggestion.Label != "bug-investigation" || suggestion.InsertText != "/skills show bug-investigation" {
		t.Fatalf("skill suggestion = %#v", suggestion)
	}
	if suggestion.Description != "Investigate known bugs before editing" {
		t.Fatalf("skill description = %q", suggestion.Description)
	}
	if !suggestion.HideCategory {
		t.Fatal("skill name suggestion should hide repeated category column")
	}

	rendered := stripANSI(m.chromeCache)
	for _, fragment := range []string{"bug-investigation", "Investigate known bugs before editing"} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("chromeCache missing %q:\n%s", fragment, rendered)
		}
	}
	if strings.Contains(rendered, "/skills show bug-investigation") {
		t.Fatalf("skill suggestions should avoid repeated command prefix:\n%s", rendered)
	}
	detail := m.selectedSlashSuggestionDetailText()
	for _, fragment := range []string{"Investigate known bugs before editing", "project", "1 scripts"} {
		if !strings.Contains(detail, fragment) {
			t.Fatalf("selected detail missing %q: %q", fragment, detail)
		}
	}
}

func TestSlashSuggestions_SkillCandidateDetailSanitizesMultilineDescription(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "multiline-detail",
				Description: "first line\nsecond\tline\x00",
				Source:      agentskills.SourceProject,
				Scripts:     []string{"scripts/repro.sh"},
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills multi")

	suggestion, ok := m.slashSuggestions.selectedSuggestion()
	if !ok {
		t.Fatal("expected skill prompt suggestion")
	}
	if got, want := suggestion.Description, "first line second line"; got != want {
		t.Fatalf("suggestion.Description = %q, want %q", got, want)
	}
	detail := m.selectedSlashSuggestionDetailText()
	if strings.ContainsAny(detail, "\n\r\t\x00") {
		t.Fatalf("selected detail should be single-line, got %q", detail)
	}
	for _, fragment := range []string{"first line second line", "project", "1 scripts"} {
		if !strings.Contains(detail, fragment) {
			t.Fatalf("selected detail missing %q: %q", fragment, detail)
		}
	}
}

func TestSlashSuggestions_SkillNameCandidateQuotesShowCommandArgument(t *testing.T) {
	skillName := `my "quoted"  skill\`
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{},
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        skillName,
				Description: "Skill with a command-sensitive name",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills show my")

	suggestion, ok := m.slashSuggestions.selectedSuggestion()
	if !ok {
		t.Fatal("expected skill name suggestion")
	}
	invocation, ok := commandruntime.Parse(suggestion.InsertText)
	if !ok {
		t.Fatalf("suggestion InsertText should parse: %q", suggestion.InsertText)
	}
	if got := invocation.Args; len(got) != 2 || got[0] != "show" || got[1] != skillName {
		t.Fatalf("parsed suggestion args = %#v, want [show %q]", got, skillName)
	}
	if !strings.Contains(suggestion.InsertText, `"my \"quoted\"  skill"\`) {
		t.Fatalf("InsertText should quote skill name, got %q", suggestion.InsertText)
	}

	agent.handledCommands[suggestion.InsertText] = true
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != suggestion.InsertText {
		t.Fatalf("handledInputs[0] = %q, want %q", got, suggestion.InsertText)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput after command = %q, want empty", got)
	}
}

func TestSlashSuggestions_EnterExecutesSkillNameCandidate(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/skills show bug-investigation": true},
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "bug-investigation",
				Description: "Investigate known bugs before editing",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills show bug")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	requireAgentDoneCmd(t, cmd)
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/skills show bug-investigation" {
		t.Fatalf("handledInputs[0] = %q, want /skills show bug-investigation", got)
	}
	if m.textInput.Value() != "" {
		t.Fatalf("textInput after command = %q, want empty", m.textInput.Value())
	}
}

func TestSlashSuggestions_SkillsPromptCandidatesAppearBeforeSubcommands(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{
				{
					Name:        "bug-investigation",
					Description: "Investigate known bugs before editing",
				},
				{
					Name:        "skill-creator",
					Description: "Create or update Codex skills",
				},
			},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills ")

	if got := len(m.slashSuggestions.suggestions); got != 5 {
		t.Fatalf("skills suggestions len = %d, want 5", got)
	}
	if got := []string{
		m.slashSuggestions.suggestions[0].Label,
		m.slashSuggestions.suggestions[1].Label,
	}; strings.Join(got, ",") != "bug-investigation,skill-creator" {
		t.Fatalf("skill labels = %#v, want skill candidates first", got)
	}
	if got := []string{
		m.slashSuggestions.suggestions[2].Label,
		m.slashSuggestions.suggestions[3].Label,
		m.slashSuggestions.suggestions[4].Label,
	}; strings.Join(got, ",") != "overview,show <name>,doctor" {
		t.Fatalf("subcommand labels = %#v, want overview/show/doctor after skills", got)
	}
	suggestion := m.slashSuggestions.suggestions[0]
	if suggestion.Label != "bug-investigation" || suggestion.InsertText != "Use the bug-investigation skill. " {
		t.Fatalf("skill prompt suggestion = %#v", suggestion)
	}
	if !suggestion.CompleteOnEnter {
		t.Fatal("skill prompt suggestion should paste into composer on Enter")
	}
	if suggestion.SubmitOnEnter {
		t.Fatal("empty-prefix skill prompt suggestion should not submit on Enter")
	}
	rendered := stripANSI(m.chromeCache)
	if !strings.Contains(rendered, "bug-investigation") ||
		!strings.Contains(rendered, "Investigate known bugs before editing") ||
		strings.Contains(rendered, "/skills bug-investigation") {
		t.Fatalf("skills suggestions should include skill names without command prefix:\n%s", rendered)
	}
	rows := m.renderSlashSuggestionRows()
	if len(rows) == 0 {
		t.Fatal("skills suggestion rows should render")
	}
	if first := stripANSI(rows[0]); !strings.HasPrefix(first, "› bug-investigation") {
		t.Fatalf("skills suggestions should not reserve an empty category column:\n%s", first)
	}
}

func TestSlashSuggestions_EnterOnSkillsSkillCandidatePastesReference(t *testing.T) {
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
	m = sendComposerRunes(m, "/skills bug")

	if suggestion, ok := m.slashSuggestions.selectedSuggestion(); !ok ||
		suggestion.Label != "bug-investigation" ||
		!suggestion.CompleteOnEnter {
		t.Fatalf("selected suggestion = %#v, %v, want /skills skill prompt candidate", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills skill candidate should paste into composer, got cmd %v", cmd)
	}
	if got := m.textInput.Value(); got != "Use the bug-investigation skill. " {
		t.Fatalf("textInput after Enter = %q, want skill prompt reference", got)
	}
	if m.slashSuggestions.visible() {
		t.Fatal("slash suggestions should close after inserting skill prompt reference")
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := len(agent.chatInputs); got != 0 {
		t.Fatalf("chatInputs length = %d, want 0", got)
	}
}

func TestSlashSuggestions_EnterOnSkillsSkillCandidateSanitizesPromptReference(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		skillCatalog: agentskills.SkillCatalog{
			Skills: []agentskills.ParsedSkill{{
				Name:        "safe\n- injected\tname\x00",
				Description: "Suspicious metadata",
			}},
		},
	}
	m := newModelWithViewport(agent)
	m = sendComposerRunes(m, "/skills safe")

	if suggestion, ok := m.slashSuggestions.selectedSuggestion(); !ok ||
		suggestion.Label != "safe\n- injected\tname\x00" ||
		!suggestion.CompleteOnEnter {
		t.Fatalf("selected suggestion = %#v, %v, want sanitized skill prompt candidate", suggestion, ok)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("Enter on /skills skill candidate should paste into composer, got cmd %v", cmd)
	}
	if got, want := m.textInput.Value(), "Use the safe - injected name skill. "; got != want {
		t.Fatalf("textInput after Enter = %q, want %q", got, want)
	}
	if strings.ContainsAny(m.textInput.Value(), "\n\r\t\x00") {
		t.Fatalf("textInput should not contain control characters: %q", m.textInput.Value())
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := len(agent.chatInputs); got != 0 {
		t.Fatalf("chatInputs length = %d, want 0", got)
	}
}

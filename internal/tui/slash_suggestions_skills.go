package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func parseSkillsShowArgument(input string) (argPrefix string, ok bool) {
	input = strings.TrimLeft(input, " \t")
	if !strings.HasPrefix(input, "/skills") {
		return "", false
	}
	rest := input[len("/skills"):]
	if rest == "" || !startsWithHorizontalSpace(rest) {
		return "", false
	}

	rest = strings.TrimLeft(rest, " \t")
	if len(rest) < len("show") || !strings.EqualFold(rest[:len("show")], "show") {
		return "", false
	}
	rest = rest[len("show"):]
	if rest == "" || !startsWithHorizontalSpace(rest) {
		return "", false
	}

	return strings.TrimLeft(rest, " \t"), true
}

func startsWithHorizontalSpace(value string) bool {
	return strings.HasPrefix(value, " ") || strings.HasPrefix(value, "\t")
}

func (m Model) skillsArgumentSuggestions(prefix string, argPrefix string) []slash.Suggestion {
	subcommands := slash.Suggestions(prefix)
	skills := m.skillPromptReferenceSuggestions(argPrefix)

	switch skillsSubcommandPriority(argPrefix, subcommands) {
	case skillsSubcommandPriorityFirst:
		return append(subcommands, skills...)
	case skillsSubcommandPrioritySuppressSkills:
		return subcommands
	default:
		return append(skills, subcommands...)
	}
}

type skillsSubcommandPriorityMode int

const (
	skillsSubcommandPriorityNone skillsSubcommandPriorityMode = iota
	skillsSubcommandPriorityFirst
	skillsSubcommandPrioritySuppressSkills
)

func skillsSubcommandPriority(argPrefix string, subcommands []slash.Suggestion) skillsSubcommandPriorityMode {
	arg := strings.ToLower(strings.TrimSpace(argPrefix))
	if arg == "" {
		return skillsSubcommandPriorityNone
	}
	for _, suggestion := range subcommands {
		if arg == skillsSubcommandToken(suggestion) {
			return skillsSubcommandPriorityFirst
		}
	}
	if arg == "list" {
		return skillsSubcommandPrioritySuppressSkills
	}
	return skillsSubcommandPriorityNone
}

func skillsSubcommandToken(suggestion slash.Suggestion) string {
	insertText := strings.TrimSpace(suggestion.InsertText)
	if insertText != "/skills" && !strings.HasPrefix(insertText, "/skills ") {
		return ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(insertText, "/skills"))
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(fields[0])
}

func (m Model) skillNameArgumentSuggestions(argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	catalog := m.currentSkillCatalog()
	suggestions := make([]slash.Suggestion, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		if !strings.HasPrefix(strings.ToLower(skill.Name), argPrefix) {
			continue
		}
		view := newSkillSuggestionView(skill)
		suggestions = append(suggestions, slash.Suggestion{
			Label:         view.name,
			InsertText:    view.showCommand,
			Description:   view.description,
			Category:      commandcatalog.CommandCategoryContext,
			CategoryLabel: "skill",
			HideCategory:  true,
			Detail:        view.detail,
			SubmitOnEnter: submitOnEnter,
		})
	}
	return suggestions
}

func (m Model) skillPromptReferenceSuggestions(argPrefix string) []slash.Suggestion {
	argPrefix = strings.ToLower(strings.TrimSpace(argPrefix))
	submitOnEnter := argPrefix != ""
	catalog := m.currentSkillCatalog()
	suggestions := make([]slash.Suggestion, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		if !strings.HasPrefix(strings.ToLower(skill.Name), argPrefix) {
			continue
		}
		view := newSkillSuggestionView(skill)
		if view.promptReference == "" {
			continue
		}
		suggestions = append(suggestions, slash.Suggestion{
			Label:           view.name,
			InsertText:      view.promptReference,
			Description:     view.description,
			Category:        commandcatalog.CommandCategoryContext,
			CategoryLabel:   "skill",
			HideCategory:    true,
			Detail:          view.detail,
			CompleteOnEnter: true,
			SubmitOnEnter:   submitOnEnter,
		})
	}
	return suggestions
}

func (m Model) currentSkillCatalog() agentskills.SkillCatalog {
	if m.skillCatalog == nil {
		return agentskills.SkillCatalog{}
	}
	return m.skillCatalog.SkillCatalog()
}

func skillPromptReferenceText(name string) string {
	name = agentskills.SanitizeCatalogPromptValue(name)
	if name == "" {
		return ""
	}
	return "Use the " + name + " skill. "
}

type skillSuggestionView struct {
	name            string
	description     string
	detail          string
	promptReference string
	showCommand     string
}

func newSkillSuggestionView(skill agentskills.ParsedSkill) skillSuggestionView {
	name := strings.TrimSpace(skill.Name)
	return skillSuggestionView{
		name:            name,
		description:     skillSuggestionSingleLine(skill.Description),
		detail:          skillCandidateDetail(skill),
		promptReference: skillPromptReferenceText(name),
		showCommand:     skillShowCommandText(name),
	}
}

func skillShowCommandText(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "/skills show " + commandruntime.QuoteArg(name)
}

func skillCandidateDetail(skill agentskills.ParsedSkill) string {
	parts := make([]string, 0, 3)
	if description := skillSuggestionSingleLine(skill.Description); description != "" {
		parts = append(parts, description)
	}
	if source := skillSuggestionSingleLine(string(skill.Source)); source != "" {
		parts = append(parts, source)
	}
	if resources := skillSuggestionSingleLine(agentskills.ResourceSummary(skill)); resources != "" {
		parts = append(parts, resources)
	}
	return strings.Join(parts, " · ")
}

func skillSuggestionSingleLine(value string) string {
	return termtext.SanitizeSingleLineANSI(strings.TrimSpace(value))
}

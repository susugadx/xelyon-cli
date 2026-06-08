package agent

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

const (
	skillsOverviewDescriptionWidth  = 116
	skillsOverviewDescriptionIndent = "    "
)

func printSkillsOverview(out io.Writer, catalog agentskills.SkillCatalog) {
	printCommandHeaderToWriter(out, "Agent Skills Overview")
	_, _ = fmt.Fprintln(out)
	green.Fprintf(out, "Skills: %d\n", len(catalog.Skills))
	dim.Fprintln(out, "Use /skills to pick and insert a skill prompt. Use /skills show <name> for full SKILL.md.")

	if len(catalog.Skills) == 0 {
		_, _ = fmt.Fprintln(out)
		dim.Fprintln(out, "No skills found.")
		printSkillDiagnosticsSummary(out, catalog)
		return
	}

	for _, group := range skillOverviewGroups(catalog.Skills) {
		if len(group.skills) == 0 {
			continue
		}
		_, _ = fmt.Fprintln(out)
		cyan.Fprintf(out, "%s (%d)\n", group.label, len(group.skills))
		for _, skill := range group.skills {
			printSkillOverviewRow(out, skill)
		}
	}

	printSkillDiagnosticsSummary(out, catalog)
}

func printSkillDiagnosticsSummary(out io.Writer, catalog agentskills.SkillCatalog) {
	if len(catalog.Diagnostics) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out)
	yellow.Fprintf(out, "Diagnostics: %d issue(s). Run /skills doctor.\n", len(catalog.Diagnostics))
}

func printSkillDetail(out io.Writer, skill agentskills.ParsedSkill) {
	printCommandHeaderToWriter(out, fmt.Sprintf("Agent Skill: %s", skill.Name))
	_, _ = fmt.Fprintln(out)

	green.Fprintf(out, "Name: %s\n", skill.Name)
	dim.Fprintf(out, "Source: %s\n", skillSourceLabel(skill.Source))
	dim.Fprintf(out, "Directory: %s\n", skill.Directory)
	if resources := agentskills.ResourceSummary(skill); resources != "" {
		dim.Fprintf(out, "Resources: %s\n", resources)
	}

	description := normalizeOverviewText(skill.Description)
	if description != "" {
		_, _ = fmt.Fprintln(out)
		cyan.Fprintln(out, "Description")
		for _, line := range wrapDisplayWidth(description, skillsOverviewDescriptionWidth) {
			_, _ = fmt.Fprintln(out, line)
		}
	}

	body := strings.TrimSpace(skill.Body)
	if body != "" {
		_, _ = fmt.Fprintln(out)
		cyan.Fprintln(out, "SKILL.md")
		_, _ = fmt.Fprintln(out, body)
	}

	printSkillResourceListings(out, skill)
}

func printSkillResourceListings(out io.Writer, skill agentskills.ParsedSkill) {
	groups := []struct {
		label string
		items []string
	}{
		{label: "scripts", items: skill.Scripts},
		{label: "references", items: skill.References},
		{label: "assets", items: skill.Assets},
	}

	hasResources := false
	for _, group := range groups {
		if len(group.items) > 0 {
			hasResources = true
			break
		}
	}
	if !hasResources {
		return
	}

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "Resources")
	for _, group := range groups {
		if len(group.items) == 0 {
			continue
		}
		dim.Fprintf(out, "%s\n", group.label)
		for _, item := range group.items {
			_, _ = fmt.Fprintf(out, "  - %s\n", item)
		}
	}
}

func skillSourceLabel(source agentskills.Source) string {
	switch source {
	case agentskills.SourceProject:
		return "project"
	case agentskills.SourceHome:
		return "home"
	case agentskills.SourceXelyon:
		return "xelyon"
	default:
		if strings.TrimSpace(string(source)) == "" {
			return "unknown"
		}
		return string(source)
	}
}

type skillOverviewGroup struct {
	label  string
	skills []agentskills.ParsedSkill
}

func skillOverviewGroups(skills []agentskills.ParsedSkill) []skillOverviewGroup {
	groups := []skillOverviewGroup{
		{label: "Project skills"},
		{label: "Home skills"},
		{label: "XELYON skills"},
		{label: "Other skills"},
	}
	for _, skill := range skills {
		switch skill.Source {
		case agentskills.SourceProject:
			groups[0].skills = append(groups[0].skills, skill)
		case agentskills.SourceHome:
			groups[1].skills = append(groups[1].skills, skill)
		case agentskills.SourceXelyon:
			groups[2].skills = append(groups[2].skills, skill)
		default:
			groups[3].skills = append(groups[3].skills, skill)
		}
	}
	return groups
}

func printSkillOverviewRow(out io.Writer, skill agentskills.ParsedSkill) {
	name := strings.TrimSpace(skill.Name)
	description := normalizeOverviewText(skill.Description)
	if description == "" {
		description = "No description"
	}

	green.Fprintf(out, "  %s\n", name)
	for _, line := range wrapDisplayWidth(description, skillsOverviewDescriptionWidth) {
		_, _ = fmt.Fprintf(out, "%s%s\n", skillsOverviewDescriptionIndent, line)
	}

	if resources := agentskills.ResourceSummary(skill); resources != "" {
		dim.Fprintf(out, "%s%s\n", skillsOverviewDescriptionIndent, resources)
	}
}

func normalizeOverviewText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func wrapDisplayWidth(value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if limit <= 0 {
		return []string{value}
	}

	var lines []string
	var current strings.Builder
	currentWidth := 0
	lastSpaceByte := -1

	for _, r := range value {
		width := runewidth.RuneWidth(r)
		if currentWidth+width > limit && current.Len() > 0 {
			if lastSpaceByte >= 0 {
				lines = append(lines, strings.TrimSpace(current.String()[:lastSpaceByte]))
				rest := strings.TrimLeft(current.String()[lastSpaceByte:], " \t")
				current.Reset()
				current.WriteString(rest)
				currentWidth = runewidth.StringWidth(rest)
			} else {
				lines = append(lines, strings.TrimSpace(current.String()))
				current.Reset()
				currentWidth = 0
			}
			lastSpaceByte = -1
		}

		current.WriteRune(r)
		currentWidth += width
		if r == ' ' || r == '\t' {
			lastSpaceByte = current.Len()
		}
	}

	if line := strings.TrimSpace(current.String()); line != "" {
		lines = append(lines, line)
	}
	return lines
}

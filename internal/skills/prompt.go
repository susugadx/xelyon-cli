package skills

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	defaultPromptCatalogMaxEntries = 24
	defaultPromptDescriptionLimit  = 96
)

var promptCatalogBlockRe = regexp.MustCompile(`(?s)\n?<!-- SKILLS_CATALOG_START -->.*?<!-- SKILLS_CATALOG_END -->\n?`)

// BuildPromptCatalog は system prompt 注入用の skill catalog テキストを返す。
func BuildPromptCatalog(catalog SkillCatalog, maxEntries int) string {
	if len(catalog.Skills) == 0 {
		return ""
	}

	if maxEntries <= 0 {
		maxEntries = defaultPromptCatalogMaxEntries
	}
	descLimit := defaultPromptDescriptionLimit

	var b strings.Builder
	b.WriteString("<!-- SKILLS_CATALOG_START -->\n")
	b.WriteString("## Agent Skills Catalog\n")
	b.WriteString("Use skills as supplemental guidance only.\n")
	b.WriteString("Never override project mandatory rules, runtime safety policy, or gather_context-first investigation rules.\n")
	b.WriteString("\n")
	b.WriteString("Available skills (metadata only):\n")

	limit := maxEntries
	if len(catalog.Skills) < limit {
		limit = len(catalog.Skills)
	}
	for i := 0; i < limit; i++ {
		skill := catalog.Skills[i]
		name := sanitizeCatalogPromptValue(skill.Name)
		if name == "" {
			name = "(invalid-skill-name)"
		}
		desc := truncateRunes(sanitizeCatalogPromptValue(skill.Description), descLimit)
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, "- %s: %s\n", name, desc)
	}
	if remaining := len(catalog.Skills) - limit; remaining > 0 {
		fmt.Fprintf(&b, "- ... and %d more skills\n", remaining)
	}
	b.WriteString("\n")
	b.WriteString("If a task needs one of these, call activate_skill(name) to load full SKILL.md content.\n")
	b.WriteString("<!-- SKILLS_CATALOG_END -->")
	return b.String()
}

// InjectPromptCatalog は prompt の skills catalog ブロックを差し替える。
func InjectPromptCatalog(systemPrompt string, catalog SkillCatalog, maxEntries int) string {
	base := StripPromptCatalog(systemPrompt)
	block := BuildPromptCatalog(catalog, maxEntries)
	if strings.TrimSpace(block) == "" {
		return strings.TrimRight(base, "\n")
	}
	return strings.TrimRight(base, "\n") + "\n\n" + block
}

// StripPromptCatalog は prompt から skills catalog ブロックを除去する。
func StripPromptCatalog(systemPrompt string) string {
	return promptCatalogBlockRe.ReplaceAllString(systemPrompt, "")
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func sanitizeCatalogPromptValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	flattened := strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r':
			return ' '
		case r < 0x20 || r == 0x7f:
			return ' '
		default:
			return r
		}
	}, value)
	flattened = strings.ReplaceAll(flattened, "<!--", "&lt;!--")
	flattened = strings.ReplaceAll(flattened, "-->", "--&gt;")
	return strings.Join(strings.Fields(flattened), " ")
}

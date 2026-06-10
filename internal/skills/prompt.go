package skills

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// DefaultPromptCatalogMaxEntries は system prompt に載せる skill metadata 件数の既定値。
	DefaultPromptCatalogMaxEntries = 24
	defaultPromptDescriptionLimit  = 96
)

var promptCatalogBlockRe = regexp.MustCompile(`(?s)\n?<!-- SKILLS_CATALOG_START -->.*?<!-- SKILLS_CATALOG_END -->\n?`)

var promptCatalogPinnedSkillNames = []string{"skill-creator"}

// BuildPromptCatalog は system prompt 注入用の skill catalog テキストを返す。
func BuildPromptCatalog(catalog SkillCatalog, maxEntries int) string {
	if len(catalog.Skills) == 0 {
		return ""
	}

	if maxEntries <= 0 {
		maxEntries = DefaultPromptCatalogMaxEntries
	}
	descLimit := defaultPromptDescriptionLimit

	var b strings.Builder
	b.WriteString("<!-- SKILLS_CATALOG_START -->\n")
	b.WriteString("## Agent Skills Catalog\n")
	b.WriteString("Use skills as supplemental guidance only.\n")
	b.WriteString("Never override project mandatory rules, runtime safety policy, or gather_context-first investigation rules.\n")
	b.WriteString("\n")
	b.WriteString("Available skills (metadata only):\n")

	entries := promptCatalogSkills(catalog.Skills, maxEntries)
	for _, skill := range entries {
		name := SanitizePromptLineValue(skill.Name)
		if name == "" {
			name = "(invalid-skill-name)"
		}
		desc := truncateRunes(SanitizeCatalogPromptValue(skill.Description), descLimit)
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(&b, "- %s: %s\n", name, desc)
	}
	if remaining := len(catalog.Skills) - len(entries); remaining > 0 {
		fmt.Fprintf(&b, "- ... and %d more skills\n", remaining)
	}
	b.WriteString("\n")
	b.WriteString("If a task needs one of these, call activate_skill(name) to load full SKILL.md content.\n")
	b.WriteString("<!-- SKILLS_CATALOG_END -->")
	return b.String()
}

func promptCatalogSkills(skills []ParsedSkill, maxEntries int) []ParsedSkill {
	if len(skills) == 0 {
		return nil
	}
	if maxEntries <= 0 {
		maxEntries = DefaultPromptCatalogMaxEntries
	}

	entries := make([]ParsedSkill, 0, minInt(maxEntries, len(skills)))
	seen := make(map[string]struct{}, maxEntries)
	appendSkill := func(skill ParsedSkill) bool {
		if len(entries) >= maxEntries {
			return false
		}
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		if key != "" {
			if _, ok := seen[key]; ok {
				return true
			}
			seen[key] = struct{}{}
		}
		entries = append(entries, skill)
		return true
	}

	for _, pinned := range promptCatalogPinnedSkillNames {
		for _, skill := range skills {
			if strings.EqualFold(strings.TrimSpace(skill.Name), pinned) {
				if !appendSkill(skill) {
					return entries
				}
				break
			}
		}
	}
	for _, skill := range skills {
		if !appendSkill(skill) {
			return entries
		}
	}
	return entries
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
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

// SanitizePromptLineValue は skill metadata を prompt / composer 用の 1 行テキストに正規化する。
func SanitizePromptLineValue(value string) string {
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
	return strings.TrimSpace(flattened)
}

// SanitizeCatalogPromptValue は skill metadata を空白正規化済みの prompt / composer 用テキストにする。
func SanitizeCatalogPromptValue(value string) string {
	return strings.Join(strings.Fields(SanitizePromptLineValue(value)), " ")
}

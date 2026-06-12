package skills

import skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"

func sanitizedSkillNames(catalog skillcatalog.SkillCatalog) []string {
	names := skillcatalog.SkillNames(catalog)
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = sanitizeSkillToolLine(name, "")
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

func sanitizeSkillToolLine(value, fallback string) string {
	value = skillcatalog.SanitizePromptLineValue(value)
	if value == "" {
		return fallback
	}
	return value
}

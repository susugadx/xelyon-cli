package skills

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed *.md
var builtinSkills embed.FS

// LoadSkill loads a skill by name.
// Priority: custom (.xelyon/skills/) > builtin (internal/skills/)
func LoadSkill(name string) (string, error) {
	// 1. カスタムスキル優先
	customPath := filepath.Join(".xelyon", "skills", name+".md")
	if content, err := os.ReadFile(customPath); err == nil {
		return string(content), nil
	}

	// 2. 組み込みスキル
	content, err := builtinSkills.ReadFile(name + ".md")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ListSkills returns available skill names.
func ListSkills() []string {
	skills := make(map[string]bool)

	// 組み込みスキル
	entries, _ := builtinSkills.ReadDir(".")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			name := strings.TrimSuffix(e.Name(), ".md")
			skills[name] = true
		}
	}

	// カスタムスキル
	customDir := filepath.Join(".xelyon", "skills")
	if entries, err := os.ReadDir(customDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				skills[name] = true
			}
		}
	}

	// map to slice
	result := make([]string, 0, len(skills))
	for name := range skills {
		result = append(result, name)
	}
	return result
}

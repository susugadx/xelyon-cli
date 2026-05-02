package skills

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Activate は指定名の skill を解決し、ツール返却用 payload を組み立てる。
func Activate(catalog SkillCatalog, name string) (ActivatedSkill, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ActivatedSkill{}, fmt.Errorf("skill name is required")
	}

	for _, skill := range catalog.Skills {
		if skill.Name != name {
			continue
		}
		payload := buildActivatedSkillPayload(skill)
		return ActivatedSkill{
			Skill:   skill,
			Payload: payload,
			Content: renderActivatedSkillPayload(payload),
		}, nil
	}

	return ActivatedSkill{}, fmt.Errorf("unknown skill: %s", name)
}

func buildActivatedSkillPayload(skill ParsedSkill) ActivatedSkillPayload {
	return ActivatedSkillPayload{
		Name:           skill.Name,
		SkillDirectory: skill.Directory,
		Scripts:        append([]string(nil), skill.Scripts...),
		References:     append([]string(nil), skill.References...),
		Assets:         append([]string(nil), skill.Assets...),
		SkillMD:        skill.Body,
	}
}

func renderActivatedSkillPayload(payload ActivatedSkillPayload) string {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to render activated skill payload: %v"}`, err)
	}
	return string(data)
}

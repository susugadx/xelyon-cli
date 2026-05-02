package skills

type SkillResourceGroup string

const (
	SkillResourceGroupScripts    SkillResourceGroup = "scripts"
	SkillResourceGroupReferences SkillResourceGroup = "references"
	SkillResourceGroupAssets     SkillResourceGroup = "assets"
)

var skillResourceGroupOrder = []SkillResourceGroup{
	SkillResourceGroupScripts,
	SkillResourceGroupReferences,
	SkillResourceGroupAssets,
}

func (g SkillResourceGroup) String() string {
	return string(g)
}

func skillResourceItems(skill ParsedSkill, group SkillResourceGroup) []string {
	switch group {
	case SkillResourceGroupScripts:
		return skill.Scripts
	case SkillResourceGroupReferences:
		return skill.References
	case SkillResourceGroupAssets:
		return skill.Assets
	default:
		return nil
	}
}

func setSkillResourceItems(skill *ParsedSkill, group SkillResourceGroup, items []string) {
	if skill == nil {
		return
	}
	switch group {
	case SkillResourceGroupScripts:
		skill.Scripts = items
	case SkillResourceGroupReferences:
		skill.References = items
	case SkillResourceGroupAssets:
		skill.Assets = items
	}
}

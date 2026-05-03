package skills

func cloneDiscoverResult(discover DiscoverResult) DiscoverResult {
	cloned := DiscoverResult{
		Roots:       append([]string(nil), discover.Roots...),
		Skills:      make([]DiscoveredSkill, len(discover.Skills)),
		Diagnostics: append([]Diagnostic(nil), discover.Diagnostics...),
	}
	copy(cloned.Skills, discover.Skills)
	return cloned
}

func cloneSkillCatalog(catalog SkillCatalog) SkillCatalog {
	cloned := SkillCatalog{
		Skills:      make([]ParsedSkill, len(catalog.Skills)),
		Diagnostics: append([]Diagnostic(nil), catalog.Diagnostics...),
	}
	for i, skill := range catalog.Skills {
		cloned.Skills[i] = cloneParsedSkill(skill)
	}
	return cloned
}

func cloneParsedSkill(skill ParsedSkill) ParsedSkill {
	cloned := skill
	for _, group := range skillResourceGroupOrder {
		items := append([]string(nil), skillResourceItems(skill, group)...)
		setSkillResourceItems(&cloned, group, items)
	}
	return cloned
}

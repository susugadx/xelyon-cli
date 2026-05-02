package agent

import (
	"strings"
	"testing"

	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestInjectSkillCatalogPrompt_MetadataOnly(t *testing.T) {
	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()

	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog {
		return agentskills.SkillCatalog{Skills: []agentskills.ParsedSkill{
			{Name: "demo", Description: "demo description", Body: "# Secret body"},
		}}
	}

	got := injectSkillCatalogPrompt("base prompt", "")
	if !strings.Contains(got, "## Agent Skills Catalog") {
		t.Fatalf("skills catalog header missing:\n%s", got)
	}
	if !strings.Contains(got, "- demo: demo description") {
		t.Fatalf("skills catalog entry missing:\n%s", got)
	}
	if !strings.Contains(got, "activate_skill(name)") {
		t.Fatalf("activate_skill guidance missing:\n%s", got)
	}
	if strings.Contains(got, "# Secret body") {
		t.Fatalf("prompt should not include SKILL.md body:\n%s", got)
	}
}

func TestInjectSkillCatalogPrompt_ReplacesExistingBlock(t *testing.T) {
	oldLoader := loadSkillCatalogForAgent
	defer func() { loadSkillCatalogForAgent = oldLoader }()

	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog {
		return agentskills.SkillCatalog{Skills: []agentskills.ParsedSkill{{Name: "one", Description: "desc"}}}
	}
	first := injectSkillCatalogPrompt("base", "")
	if strings.Count(first, "SKILLS_CATALOG_START") != 1 {
		t.Fatalf("first prompt should contain one skills block:\n%s", first)
	}

	loadSkillCatalogForAgent = func(_ string) agentskills.SkillCatalog {
		return agentskills.SkillCatalog{Skills: []agentskills.ParsedSkill{{Name: "two", Description: "desc2"}}}
	}
	second := injectSkillCatalogPrompt(first, "")
	if strings.Contains(second, "- one: desc") {
		t.Fatalf("old skills entry should be replaced:\n%s", second)
	}
	if !strings.Contains(second, "- two: desc2") {
		t.Fatalf("new skills entry missing:\n%s", second)
	}
	if strings.Count(second, "SKILLS_CATALOG_START") != 1 {
		t.Fatalf("second prompt should contain one skills block:\n%s", second)
	}
}

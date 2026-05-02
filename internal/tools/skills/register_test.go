package skills

import (
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestActivateSkillTool_Run(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{
			{
				Name:        "demo",
				Description: "desc",
				Body:        "# body",
				Directory:   "/tmp/demo",
				Scripts:     []string{"scripts/run.sh"},
			},
		}}
	}

	tool := &ActivateSkillTool{}
	got, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{"name": "demo"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatalf("Run() change = %#v, want nil", change)
	}
	if !strings.Contains(got, `"name": "demo"`) {
		t.Fatalf("Run() output should include skill name in JSON payload:\n%s", got)
	}
	if !strings.Contains(got, `"skill_md": "# body"`) {
		t.Fatalf("Run() output should include skill_md in JSON payload:\n%s", got)
	}
}

func TestActivateSkillTool_ParametersAreStaticAndNoEnum(t *testing.T) {
	oldLoader := loadCatalogForTool
	defer func() { loadCatalogForTool = oldLoader }()

	loadCalls := 0
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		loadCalls++
		return skillcatalog.SkillCatalog{}
	}

	tool := &ActivateSkillTool{}
	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing: %#v", params)
	}
	nameProp, ok := properties["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("name property missing: %#v", properties)
	}
	description, ok := nameProp["description"].(string)
	if !ok {
		t.Fatalf("name description missing or invalid: %#v", nameProp)
	}
	if !strings.Contains(description, "Skill name to activate.") {
		t.Fatalf("description should be static, got: %q", description)
	}
	if _, hasEnum := nameProp["enum"]; hasEnum {
		t.Fatalf("name property should not pin enum (invocation cwd can differ): %#v", nameProp)
	}
	if loadCalls != 0 {
		t.Fatalf("Parameters() should not load catalog, loadCalls=%d", loadCalls)
	}
}

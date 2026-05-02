package skills

import (
	"fmt"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

var loadCatalogForTool = func(invocationCWD string) skillcatalog.SkillCatalog {
	return skillcatalog.LoadCatalogForInvocationCWD(invocationCWD)
}

// ActivateSkillTool は skill 本文を動的に読み出す read-only ツール。
type ActivateSkillTool struct{}

func (t *ActivateSkillTool) Name() string { return "activate_skill" }

func (t *ActivateSkillTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ActivateSkillTool) Parameters() map[string]interface{} {
	nameProperty := map[string]interface{}{
		"type":        "string",
		"description": "Skill name to activate. If unknown, run /skills list or ask the user to provide an exact name.",
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": nameProperty,
		},
		"required":             []string{"name"},
		"additionalProperties": false,
	}
}

func (t *ActivateSkillTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	name := strings.TrimSpace(args["name"])
	if name == "" {
		return "Error: skill name is required", nil, nil
	}

	catalog := loadCatalogForTool(execCtx.InvocationCWD)
	activated, err := skillcatalog.Activate(catalog, name)
	if err != nil {
		available := strings.Join(skillcatalog.SkillNames(catalog), ", ")
		if available == "" {
			return "Error: no skills are available", nil, nil
		}
		return fmt.Sprintf("Error: %v. Available skills: %s", err, available), nil, nil
	}

	return activated.Content, nil, nil
}

// RegisterTools は skills ツール群を登録する。
func RegisterTools(registry *tools.Registry) {
	registry.Register(&ActivateSkillTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}

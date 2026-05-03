package skills

import (
	"path/filepath"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/dev"
)

type skillToolDependencies struct {
	loadCatalog       func(invocationCWD string) skillcatalog.SkillCatalog
	resolveScriptPath func(skill skillcatalog.ParsedSkill, scriptPath string) (string, error)
	executeScript     func(execCtx tools.ExecutionContext, command string) string
}

func defaultSkillToolDependencies() skillToolDependencies {
	return skillToolDependencies{
		loadCatalog: skillcatalog.LoadCatalogForInvocationCWD,
		resolveScriptPath: func(skill skillcatalog.ParsedSkill, scriptPath string) (string, error) {
			return skillcatalog.ResolveScriptPath(skill, scriptPath)
		},
		executeScript: func(execCtx tools.ExecutionContext, command string) string {
			return dev.ExecuteBashWithContextAndPromptIOAndConfig(execCtx.EffectiveContext(), execCtx.PromptIO(), execCtx.EffectiveConfig(), command)
		},
	}
}

var skillToolDeps = defaultSkillToolDependencies()

var loadCatalogForTool = func(invocationCWD string) skillcatalog.SkillCatalog {
	return skillToolDeps.loadCatalog(invocationCWD)
}

var resolveScriptPathForTool = func(skill skillcatalog.ParsedSkill, scriptPath string) (string, error) {
	return skillToolDeps.resolveScriptPath(skill, scriptPath)
}

var executeSkillScriptCommand = func(execCtx tools.ExecutionContext, command string) string {
	return skillToolDeps.executeScript(execCtx, command)
}

func findSkillByName(catalog skillcatalog.SkillCatalog, name string) (skillcatalog.ParsedSkill, bool) {
	for _, skill := range catalog.Skills {
		if skill.Name == name {
			return skill, true
		}
	}
	return skillcatalog.ParsedSkill{}, false
}

func normalizeRequestedScriptPath(script string) string {
	script = strings.TrimSpace(script)
	slashed := filepath.ToSlash(script)
	if strings.HasPrefix(slashed, "scripts/") {
		return strings.TrimPrefix(slashed, "scripts/")
	}
	return script
}

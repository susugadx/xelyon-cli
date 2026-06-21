package skills

import (
	"fmt"
	"strings"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RunSkillScriptTool は skill scripts 配下のスクリプトを既存 bash 経路で実行する薄いラッパー。
type RunSkillScriptTool struct{}

func (t *RunSkillScriptTool) Name() string { return "run_skill_script" }

func (t *RunSkillScriptTool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *RunSkillScriptTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	request, err := parseRunSkillScriptRequest(args)
	if err != nil {
		return "Error: " + err.Error(), nil, nil
	}

	catalog := loadCatalogForTool(execCtx.InvocationCWD)
	skill, ok := findSkillByName(catalog, request.skillName)
	if !ok {
		return runSkillUnknownSkillMessage(catalog, request.skillName), nil, nil
	}

	normalizedArgs, err := normalizeRunSkillScriptArgs(request.args, request.argsJSON)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}

	resolvedPath, err := resolveScriptPathForTool(skill, normalizeRequestedScriptPath(request.scriptPath))
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}
	if err := validateResolvedSkillScriptPath(skill, resolvedPath); err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}

	command, err := buildSkillScriptCommand(resolvedPath, normalizedArgs)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}

	return executeSkillScriptCommand(execCtx, command), nil, nil
}

func runSkillUnknownSkillMessage(catalog skillcatalog.SkillCatalog, skillName string) string {
	skillName = sanitizeSkillToolLine(skillName, "(invalid-skill-name)")
	available := strings.Join(sanitizedSkillNames(catalog), ", ")
	if available == "" {
		return fmt.Sprintf("Error: unknown skill: %s", skillName)
	}
	return fmt.Sprintf("Error: unknown skill: %s. Available skills: %s", skillName, available)
}

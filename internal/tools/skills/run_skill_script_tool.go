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
	return tools.ToolDescriptions[t.Name()]
}

func (t *RunSkillScriptTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"skill": map[string]interface{}{
				"type":        "string",
				"description": "Skill name from the current catalog.",
			},
			"script": map[string]interface{}{
				"type":        "string",
				"description": "Script path under the skill scripts directory.",
			},
			"args": map[string]interface{}{
				"type":        "string",
				"description": "Legacy simple args string (space-delimited tokens only). Prefer args_json for quoted values or shell metacharacters.",
			},
			"args_json": map[string]interface{}{
				"type":        "string",
				"description": `JSON array of string arguments. Preferred over args. Example: ["--name","test user","--json"].`,
			},
		},
		"required":             []string{"skill", "script"},
		"additionalProperties": false,
	}
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

	command, err := buildSkillScriptCommand(resolvedPath, normalizedArgs)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil, nil
	}

	return executeSkillScriptCommand(execCtx, command), nil, nil
}

type runSkillScriptRequest struct {
	skillName  string
	scriptPath string
	args       string
	argsJSON   string
}

func parseRunSkillScriptRequest(args map[string]string) (runSkillScriptRequest, error) {
	skillName := strings.TrimSpace(args["skill"])
	if skillName == "" {
		return runSkillScriptRequest{}, fmt.Errorf("skill name is required")
	}
	scriptPath := strings.TrimSpace(args["script"])
	if scriptPath == "" {
		return runSkillScriptRequest{}, fmt.Errorf("script path is required")
	}
	legacyArgs := strings.TrimSpace(args["args"])
	argsJSON := strings.TrimSpace(args["args_json"])
	if legacyArgs != "" && argsJSON != "" {
		return runSkillScriptRequest{}, fmt.Errorf("use either args_json or args, not both")
	}

	return runSkillScriptRequest{
		skillName:  skillName,
		scriptPath: scriptPath,
		args:       legacyArgs,
		argsJSON:   argsJSON,
	}, nil
}

func runSkillUnknownSkillMessage(catalog skillcatalog.SkillCatalog, skillName string) string {
	available := strings.Join(skillcatalog.SkillNames(catalog), ", ")
	if available == "" {
		return fmt.Sprintf("Error: unknown skill: %s", skillName)
	}
	return fmt.Sprintf("Error: unknown skill: %s. Available skills: %s", skillName, available)
}

package skills

import (
	"fmt"
	"strings"
)

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

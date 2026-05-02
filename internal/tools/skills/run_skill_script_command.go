package skills

import (
	"fmt"
	"path/filepath"
	"strings"
)

type skillScriptRunner struct {
	command string
}

var skillScriptRunnersByExt = map[string]skillScriptRunner{
	".py":  {command: "python3"},
	".sh":  {command: "bash"},
	".js":  {command: "node"},
	".mjs": {command: "node"},
}

func buildSkillScriptCommand(resolvedScriptPath string, rawArgs string) (string, error) {
	runner, ext, ok := resolveSkillScriptRunner(resolvedScriptPath)
	if !ok {
		if ext == "" {
			return "", fmt.Errorf("unsupported script extension: (none)")
		}
		return "", fmt.Errorf("unsupported script extension: %s", ext)
	}

	command := runner.command + " " + shellQuote(resolvedScriptPath)
	trimmedArgs := strings.TrimSpace(rawArgs)
	if trimmedArgs == "" {
		return command, nil
	}
	return command + " " + trimmedArgs, nil
}

func resolveSkillScriptRunner(path string) (skillScriptRunner, string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	runner, ok := skillScriptRunnersByExt[ext]
	return runner, ext, ok
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

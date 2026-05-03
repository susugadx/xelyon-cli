package skills

import (
	"encoding/json"
	"errors"
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

const invalidArgsJSONArrayError = "invalid args_json: expected JSON array of strings"

func buildSkillScriptCommand(resolvedScriptPath string, argv []string) (string, error) {
	runner, ext, ok := resolveSkillScriptRunner(resolvedScriptPath)
	if !ok {
		if ext == "" {
			return "", fmt.Errorf("unsupported script extension: (none)")
		}
		return "", fmt.Errorf("unsupported script extension: %s", ext)
	}

	parts := []string{runner.command, shellQuote(resolvedScriptPath)}
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " "), nil
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

func normalizeRunSkillScriptArgs(legacyArgs, argsJSON string) ([]string, error) {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON != "" {
		return parseSkillScriptArgsJSON(argsJSON)
	}
	return parseLegacySkillScriptArgs(legacyArgs)
}

func parseSkillScriptArgsJSON(raw string) ([]string, error) {
	var decoded []any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, invalidArgsJSONTypeError()
	}
	argv := make([]string, 0, len(decoded))
	for i, item := range decoded {
		value, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("invalid args_json: argument at index %d must be a string", i)
		}
		if strings.ContainsAny(value, "\n\r\x00") {
			return nil, fmt.Errorf("invalid args_json: argument at index %d contains unsupported control characters", i)
		}
		argv = append(argv, value)
	}
	return argv, nil
}

func parseLegacySkillScriptArgs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if containsUnsafeLegacySkillScriptArgs(raw) {
		return nil, fmt.Errorf("unsafe legacy args; use args_json for quoted values or shell metacharacters")
	}
	return strings.Fields(raw), nil
}

func containsUnsafeLegacySkillScriptArgs(raw string) bool {
	return strings.ContainsAny(raw, ";&|<>$`\n\r\x00'\"()")
}

func invalidArgsJSONTypeError() error {
	return errors.New(invalidArgsJSONArrayError)
}

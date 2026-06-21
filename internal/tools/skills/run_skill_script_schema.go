package skills

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

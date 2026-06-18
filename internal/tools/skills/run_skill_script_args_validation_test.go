package skills

import (
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRunSkillScriptTool_Run_ArgsAndArgsJSONMutualExclusive(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":     "demo",
		"script":    "safe.sh",
		"args":      "--name test",
		"args_json": `["--name","test"]`,
	})
	if got != "Error: use either args_json or args, not both" {
		t.Fatalf("Run() output = %q", got)
	}
}

func TestRunSkillScriptTool_Run_InvalidArgsJSONRejectsBeforeExecution(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	executeCalls := 0
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		executeCalls++
		return command
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":     "demo",
		"script":    "safe.sh",
		"args_json": "{bad",
	})
	if got != "Error: invalid args_json: expected JSON array of strings" {
		t.Fatalf("Run() output = %q", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

func TestRunSkillScriptTool_Run_ArgsJSONNonStringRejectsBeforeExecution(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	executeCalls := 0
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		executeCalls++
		return command
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":     "demo",
		"script":    "safe.sh",
		"args_json": `["ok",123]`,
	})
	if got != "Error: invalid args_json: argument at index 1 must be a string" {
		t.Fatalf("Run() output = %q", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

func TestRunSkillScriptTool_Run_LegacyShellMetacharRejectsBeforeExecution(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	executeCalls := 0
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		executeCalls++
		return command
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "safe.sh",
		"args":   "; rm -rf /",
	})
	if got != "Error: unsafe legacy args; use args_json for quoted values or shell metacharacters" {
		t.Fatalf("Run() output = %q", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

func TestRunSkillScriptTool_Run_LegacyQuoteRejectsBeforeExecution(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	executeCalls := 0
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		executeCalls++
		return command
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "safe.sh",
		"args":   "--name 'test user'",
	})
	if got != "Error: unsafe legacy args; use args_json for quoted values or shell metacharacters" {
		t.Fatalf("Run() output = %q", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

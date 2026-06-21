package skills

import (
	"path/filepath"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRunSkillScriptTool_Run_SafeScriptPassesToExecutionPath(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	var gotCommand string
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		gotCommand = command
		return "ok"
	}

	tool := &RunSkillScriptTool{}
	got, change, err := tool.Run(tools.ExecutionContext{
		InvocationCWD: skillWorkspaceRoot(skill),
	}, map[string]string{
		"skill":  "demo",
		"script": "scripts/safe.sh",
		"args":   "--name test --json",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatalf("Run() change = %#v, want nil", change)
	}
	if got != "ok" {
		t.Fatalf("Run() output = %q, want %q", got, "ok")
	}

	expected := "bash " + shellQuote(filepath.Clean(scriptPath)) + " '--name' 'test' '--json'"
	if gotCommand != expected {
		t.Fatalf("forwarded command = %q, want %q", gotCommand, expected)
	}
}

func TestRunSkillScriptTool_Run_ArgsJSONQuotedAsArgv(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	var gotCommand string
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		gotCommand = command
		return "ok"
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{
		InvocationCWD: skillWorkspaceRoot(skill),
	}, map[string]string{
		"skill":     "demo",
		"script":    "safe.sh",
		"args_json": `["--name","test user","--json"]`,
	})
	if got != "ok" {
		t.Fatalf("Run() output = %q, want ok", got)
	}
	expected := "bash " + shellQuote(filepath.Clean(scriptPath)) + " '--name' 'test user' '--json'"
	if gotCommand != expected {
		t.Fatalf("forwarded command = %q, want %q", gotCommand, expected)
	}
}

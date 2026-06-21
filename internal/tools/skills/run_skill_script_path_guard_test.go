package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRunSkillScriptTool_Run_UnknownSkill(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{{Name: "demo"}}}
	}

	var executeCalls int
	executeSkillScriptCommand = func(_ tools.ExecutionContext, _ string) string {
		executeCalls++
		return "should-not-run"
	}

	tool := &RunSkillScriptTool{}
	got, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "missing",
		"script": "safe.sh",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if change != nil {
		t.Fatalf("Run() change = %#v, want nil", change)
	}
	if !strings.Contains(got, "Error: unknown skill: missing") {
		t.Fatalf("Run() output = %q, want unknown skill error", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

func TestRunSkillScriptTool_Run_UnknownScript(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	mustWriteRunSkillFile(t, filepath.Join(skill.Directory, "scripts", "safe.sh"), "echo safe\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	var executeCalls int
	executeSkillScriptCommand = func(_ tools.ExecutionContext, _ string) string {
		executeCalls++
		return "should-not-run"
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "missing.sh",
	})
	if !strings.Contains(got, "Error: script not found") {
		t.Fatalf("Run() output = %q, want script not found error", got)
	}
	if executeCalls != 0 {
		t.Fatalf("executeSkillScriptCommand calls = %d, want 0", executeCalls)
	}
}

func TestRunSkillScriptTool_Run_TraversalReject(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "../outside.sh",
	})
	if !strings.Contains(got, "Error: script path escapes scripts directory") {
		t.Fatalf("Run() output = %q, want traversal rejection", got)
	}
}

func TestRunSkillScriptTool_Run_AbsolutePathReject(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "/tmp/outside.sh",
	})
	if !strings.Contains(got, "Error: absolute script path is not allowed") {
		t.Fatalf("Run() output = %q, want absolute-path rejection", got)
	}
}

func TestRunSkillScriptTool_Run_SymlinkEscapeReject(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	outsidePath := filepath.Join(t.TempDir(), "outside.sh")
	mustWriteRunSkillFile(t, outsidePath, "echo outside\n")
	linkPath := filepath.Join(skill.Directory, "scripts", "escape.sh")
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "escape.sh",
	})
	if !strings.Contains(got, "Error: script symlink escapes skill scripts directory") {
		t.Fatalf("Run() output = %q, want symlink-escape rejection", got)
	}
}

func TestRunSkillScriptTool_Run_UnsupportedExtensionReject(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	mustWriteRunSkillFile(t, filepath.Join(skill.Directory, "scripts", "task.txt"), "noop\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{
		InvocationCWD: skillWorkspaceRoot(skill),
	}, map[string]string{
		"skill":  "demo",
		"script": "task.txt",
	})
	if !strings.Contains(got, "Error: unsupported script extension: .txt") {
		t.Fatalf("Run() output = %q, want unsupported-extension rejection", got)
	}
}

func TestRunSkillScriptTool_Run_AllowsParentProjectSkillFromNestedInvocation(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	workspace := t.TempDir()
	skillName := "run-skill-script-nested-invocation-test-only"
	skill := makeScriptSkillInRoot(t, workspace, skillName)
	mustWriteRunSkillSkillDefinition(t, skill.Directory, skillName)
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")
	nestedInvocation := filepath.Join(workspace, "pkg", "nested")
	if err := os.MkdirAll(nestedInvocation, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	var gotCommand string
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		gotCommand = command
		return "ok"
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{
		InvocationCWD: nestedInvocation,
	}, map[string]string{
		"skill":  skillName,
		"script": "safe.sh",
	})

	if got != "ok" {
		t.Fatalf("Run() output = %q, want ok", got)
	}
	expected := "bash " + shellQuote(filepath.Clean(scriptPath))
	if gotCommand != expected {
		t.Fatalf("forwarded command = %q, want %q", gotCommand, expected)
	}
}

func TestRunSkillScriptTool_Run_AllowsHomeSkillOutsideWorkspace(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	homeDir := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", homeDir)

	skillName := "run-skill-script-home-invocation-test-only"
	skill := makeScriptSkillInRoot(t, homeDir, skillName)
	mustWriteRunSkillSkillDefinition(t, skill.Directory, skillName)
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")

	workspace := filepath.Join(t.TempDir(), "workspace")
	nestedInvocation := filepath.Join(workspace, "pkg", "nested")
	if err := os.MkdirAll(nestedInvocation, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	var gotCommand string
	executeSkillScriptCommand = func(_ tools.ExecutionContext, command string) string {
		gotCommand = command
		return "ok"
	}

	tool := &RunSkillScriptTool{}
	got, _, _ := tool.Run(tools.ExecutionContext{
		InvocationCWD: nestedInvocation,
	}, map[string]string{
		"skill":  skillName,
		"script": "safe.sh",
	})

	if got != "ok" {
		t.Fatalf("Run() output = %q, want ok", got)
	}
	expected := "bash " + shellQuote(filepath.Clean(scriptPath))
	if gotCommand != expected {
		t.Fatalf("forwarded command = %q, want %q", gotCommand, expected)
	}
}

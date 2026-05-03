package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRunSkillScriptTool_ParametersIncludeArgsJSONAndLegacyArgs(t *testing.T) {
	tool := &RunSkillScriptTool{}
	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing: %#v", params)
	}

	argsProp, ok := properties["args"].(map[string]interface{})
	if !ok {
		t.Fatalf("args property missing: %#v", properties)
	}
	argsDesc, _ := argsProp["description"].(string)
	if !strings.Contains(argsDesc, "Legacy simple args") {
		t.Fatalf("args description should mark legacy usage, got: %q", argsDesc)
	}

	argsJSONProp, ok := properties["args_json"].(map[string]interface{})
	if !ok {
		t.Fatalf("args_json property missing: %#v", properties)
	}
	argsJSONDesc, _ := argsJSONProp["description"].(string)
	if !strings.Contains(argsJSONDesc, "JSON array of string arguments") {
		t.Fatalf("args_json description should describe JSON argv, got: %q", argsJSONDesc)
	}
}

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

func TestRunSkillScriptTool_Run_DoesNotBypassBashPolicyOrConfirmation(t *testing.T) {
	restore := stubRunSkillScriptDependencies(t)
	defer restore()

	skill := makeScriptSkill(t, "demo")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteRunSkillFile(t, scriptPath, "echo safe\n")
	loadCatalogForTool = func(_ string) skillcatalog.SkillCatalog {
		return skillcatalog.SkillCatalog{Skills: []skillcatalog.ParsedSkill{skill}}
	}

	t.Run("confirmation rejection is respected", func(t *testing.T) {
		origSimple := common.SimpleConfirm
		origSimpleWithIO := common.SimpleConfirmWithIO
		common.SimpleConfirm = func(_ string) bool { return false }
		common.SimpleConfirmWithIO = func(_ ui.PromptIO, _ string) bool { return false }
		t.Cleanup(func() {
			common.SimpleConfirm = origSimple
			common.SimpleConfirmWithIO = origSimpleWithIO
		})

		_ = os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
		t.Cleanup(func() {
			_ = os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		})

		var out bytes.Buffer
		tool := &RunSkillScriptTool{}
		got, _, _ := tool.Run(tools.ExecutionContext{
			InvocationCWD: skillWorkspaceRoot(skill),
			Stdout:        &out,
			Stderr:        &out,
		}, map[string]string{
			"skill":  "demo",
			"script": "safe.sh",
		})
		if got != "Cancelled by user" {
			t.Fatalf("Run() output = %q, want %q", got, "Cancelled by user")
		}
	})
}

func stubRunSkillScriptDependencies(t *testing.T) func() {
	t.Helper()
	oldLoader := loadCatalogForTool
	oldResolver := resolveScriptPathForTool
	oldExecutor := executeSkillScriptCommand
	return func() {
		loadCatalogForTool = oldLoader
		resolveScriptPathForTool = oldResolver
		executeSkillScriptCommand = oldExecutor
	}
}

func makeScriptSkill(t *testing.T, name string) skillcatalog.ParsedSkill {
	t.Helper()
	root := t.TempDir()
	return makeScriptSkillInRoot(t, root, name)
}

func makeScriptSkillInRoot(t *testing.T, root, name string) skillcatalog.ParsedSkill {
	t.Helper()
	skillDir := filepath.Join(root, ".agents", "skills", name)
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return skillcatalog.ParsedSkill{
		Name:      name,
		Directory: skillDir,
		Scripts:   []string{"scripts/safe.sh"},
	}
}

func mustWriteRunSkillFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func mustWriteRunSkillSkillDefinition(t *testing.T, skillDir, skillName string) {
	t.Helper()
	content := strings.Join([]string{
		"---",
		"name: " + skillName,
		"description: run skill test",
		"---",
		"# run skill test",
		"",
	}, "\n")
	mustWriteRunSkillFile(t, filepath.Join(skillDir, "SKILL.md"), content)
}

func skillWorkspaceRoot(skill skillcatalog.ParsedSkill) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(skill.Directory)))
}

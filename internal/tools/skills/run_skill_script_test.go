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
	got, _, _ := tool.Run(tools.ExecutionContext{}, map[string]string{
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
	got, change, err := tool.Run(tools.ExecutionContext{}, map[string]string{
		"skill":  "demo",
		"script": "scripts/safe.sh",
		"args":   "--name test",
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

	expected := "bash " + shellQuote(filepath.Clean(scriptPath)) + " --name test"
	if gotCommand != expected {
		t.Fatalf("forwarded command = %q, want %q", gotCommand, expected)
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

	t.Run("dangerous args blocked by bash policy", func(t *testing.T) {
		var out bytes.Buffer
		tool := &RunSkillScriptTool{}
		got, _, _ := tool.Run(tools.ExecutionContext{
			Stdout: &out,
			Stderr: &out,
		}, map[string]string{
			"skill":  "demo",
			"script": "safe.sh",
			"args":   "; rm -rf /",
		})
		if !strings.Contains(got, "Error:") || (!strings.Contains(got, "blocked") && !strings.Contains(got, "injection")) {
			t.Fatalf("Run() output = %q, want bash policy block", got)
		}
	})

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
			Stdout: &out,
			Stderr: &out,
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

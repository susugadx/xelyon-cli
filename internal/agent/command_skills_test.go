package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

func TestHandleSkillsCommand_ListAndShow(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: demo\ndescription: demo description\n---\n# Demo\nUse demo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(skillDir, "scripts", "run.sh"):        "#!/usr/bin/env bash\necho run\n",
		filepath.Join(skillDir, "references", "guide.md"):   "guide",
		filepath.Join(skillDir, "assets", "screenshot.txt"): "asset",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}
	for _, fragment := range []string{"Agent Skills Overview", "Project skills", "demo", "demo description"} {
		if !strings.Contains(out.String(), fragment) {
			t.Fatalf("/skills overview output missing %q:\n%s", fragment, out.String())
		}
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills list", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills list should be handled as overview alias")
	}
	if !strings.Contains(out.String(), "demo") {
		t.Fatalf("/skills list alias output missing catalog entry:\n%s", out.String())
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills show demo", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills show should be handled")
	}
	got := out.String()
	for _, fragment := range []string{"Agent Skill: demo", "Name: demo", "Source: project", "Resources: 1 scripts, 1 references, 1 assets", "Description", "demo description", "SKILL.md", "# Demo", "Use demo.", "Resources", "scripts", "scripts/run.sh", "references", "references/guide.md", "assets", "assets/screenshot.txt"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills show output missing %q:\n%s", fragment, got)
		}
	}
	for _, fragment := range []string{`"name": "demo"`, `"skill_md"`, `\nUse demo.`} {
		if strings.Contains(got, fragment) {
			t.Fatalf("/skills show output should be human-readable, found raw JSON fragment %q:\n%s", fragment, got)
		}
	}
}

func TestHandleSkillsCommand_ShowQuotedSkillName(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillName := `my "quoted"  skill\`
	skillDir := filepath.Join(".agents", "skills", "quoted-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "---\nname: 'my \"quoted\"  skill\\'\ndescription: quoted skill description\n---\n# Quoted\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	command := "/skills show " + commandruntime.QuoteArg(skillName)
	if !handleSpecialCommandForSurface(command, agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatalf("%s should be handled", command)
	}

	got := out.String()
	for _, fragment := range []string{"Agent Skill: " + skillName, "quoted skill description", "# Quoted"} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills show output missing %q:\n%s", fragment, got)
		}
	}
}

func TestHandleSkillsCommand_OverviewWrapsDescriptionsWithoutTruncating(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	skillDir := filepath.Join(".agents", "skills", "long-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	description := strings.Repeat("長い説明文を折り返して表示する。", 10) + "最後まで表示される"
	content := "---\nname: long-skill\ndescription: " + description + "\n---\n# Long\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}

	got := out.String()
	if !strings.Contains(got, "最後まで表示される") {
		t.Fatalf("/skills overview should not truncate long descriptions:\n%s", got)
	}
	if strings.Contains(got, "…") {
		t.Fatalf("/skills overview should wrap instead of ellipsizing:\n%s", got)
	}
	if !strings.Contains(got, "\n    ") {
		t.Fatalf("/skills overview should render wrapped description lines with indentation:\n%s", got)
	}
}

func TestHandleSkillsCommand_OverviewShowsDiagnosticsWhenNoSkillsParse(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	brokenDir := filepath.Join(".agents", "skills", "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	broken := "---\nname: broken\n---\n# missing description\n"
	if err := os.WriteFile(filepath.Join(brokenDir, "SKILL.md"), []byte(broken), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills overview", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills overview should be handled")
	}
	got := out.String()
	for _, fragment := range []string{"Agent Skills Overview", "Skills: 0", "No skills found.", "Diagnostics: 1 issue(s). Run /skills doctor."} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("/skills overview output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "Project skills") {
		t.Fatalf("/skills overview should not render skill groups when no skills parse:\n%s", got)
	}
}

func TestHandleSkillsCommand_DoctorShowsDiagnostics(t *testing.T) {
	disableColors(t)
	agent, out := newSurfaceDispatchTestAgent(t)

	brokenDir := filepath.Join(".agents", "skills", "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	broken := "---\nname: broken\n---\n# missing description\n"
	if err := os.WriteFile(filepath.Join(brokenDir, "SKILL.md"), []byte(broken), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !handleSpecialCommandForSurface("/skills doctor", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills doctor should be handled")
	}
	if !strings.Contains(out.String(), "parse_skill_failed") {
		t.Fatalf("/skills doctor output should include parse diagnostics:\n%s", out.String())
	}
}

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
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

	if !handleSpecialCommandForSurface("/skills list", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills list should be handled")
	}
	if !strings.Contains(out.String(), "- demo: demo description") {
		t.Fatalf("/skills list output missing catalog entry:\n%s", out.String())
	}

	out.Reset()
	if !handleSpecialCommandForSurface("/skills show demo", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/skills show should be handled")
	}
	if !strings.Contains(out.String(), `"name": "demo"`) {
		t.Fatalf("/skills show output missing JSON payload:\n%s", out.String())
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

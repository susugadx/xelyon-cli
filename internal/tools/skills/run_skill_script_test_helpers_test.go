package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

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

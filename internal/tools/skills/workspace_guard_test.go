package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillcatalog "github.com/susugadx/xelyon-cli/internal/skills"
)

func TestValidateResolvedSkillScriptPath_AllowsProjectSkillScript(t *testing.T) {
	skill := makeBoundarySkill(t, filepath.Join(t.TempDir(), "repo"), "project-skill")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteSkillScriptBoundaryFile(t, scriptPath, "echo safe\n")

	resolvedPath, err := resolveScriptPathForTool(skill, "safe.sh")
	if err != nil {
		t.Fatalf("resolveScriptPathForTool() error = %v", err)
	}

	if err := validateResolvedSkillScriptPath(skill, resolvedPath); err != nil {
		t.Fatalf("validateResolvedSkillScriptPath() error = %v", err)
	}
}

func TestValidateResolvedSkillScriptPath_AllowsHomeSkillScript(t *testing.T) {
	homeRoot := filepath.Join(t.TempDir(), "home")
	skill := makeBoundarySkill(t, homeRoot, "home-skill")
	scriptPath := filepath.Join(skill.Directory, "scripts", "safe.sh")
	mustWriteSkillScriptBoundaryFile(t, scriptPath, "echo safe\n")

	resolvedPath, err := resolveScriptPathForTool(skill, "safe.sh")
	if err != nil {
		t.Fatalf("resolveScriptPathForTool() error = %v", err)
	}

	if err := validateResolvedSkillScriptPath(skill, resolvedPath); err != nil {
		t.Fatalf("validateResolvedSkillScriptPath() error = %v", err)
	}
}

func TestValidateResolvedSkillScriptPath_RejectsPathOutsideSkillScriptsDirectory(t *testing.T) {
	skill := makeBoundarySkill(t, filepath.Join(t.TempDir(), "repo"), "target-skill")
	otherSkill := makeBoundarySkill(t, filepath.Join(t.TempDir(), "repo"), "other-skill")

	insideSkillButOutsideScripts := filepath.Join(skill.Directory, "SKILL.md")
	mustWriteSkillScriptBoundaryFile(t, insideSkillButOutsideScripts, "---\nname: target-skill\ndescription: desc\n---\n")

	otherSkillScript := filepath.Join(otherSkill.Directory, "scripts", "other.sh")
	mustWriteSkillScriptBoundaryFile(t, otherSkillScript, "echo other\n")

	tmpScript := filepath.Join(t.TempDir(), "tmp", "foo.sh")
	mustWriteSkillScriptBoundaryFile(t, tmpScript, "echo tmp\n")

	sshScript := filepath.Join(t.TempDir(), ".ssh", "foo.sh")
	mustWriteSkillScriptBoundaryFile(t, sshScript, "echo ssh\n")

	tests := []struct {
		name string
		path string
	}{
		{name: "tmp", path: tmpScript},
		{name: "ssh", path: sshScript},
		{name: "skill root", path: filepath.Join(skill.Directory, "README.md")},
		{name: "scripts outside", path: otherSkillScript},
		{name: "inside skill but outside scripts", path: insideSkillButOutsideScripts},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResolvedSkillScriptPath(skill, tt.path)
			if err == nil {
				t.Fatal("validateResolvedSkillScriptPath() error = nil, want boundary rejection")
			}
			if !strings.Contains(err.Error(), runSkillScriptOutsideSkillScriptsDirectoryError) {
				t.Fatalf("validateResolvedSkillScriptPath() error = %v, want %q", err, runSkillScriptOutsideSkillScriptsDirectoryError)
			}
		})
	}
}

func TestValidateResolvedSkillScriptPath_ResolveScriptPathRejectsTraversal(t *testing.T) {
	skill := makeBoundarySkill(t, filepath.Join(t.TempDir(), "repo"), "target-skill")

	if _, err := resolveScriptPathForTool(skill, "../escape.sh"); err == nil {
		t.Fatal("resolveScriptPathForTool(../escape.sh) error = nil, want traversal rejection")
	}
}

func TestValidateResolvedSkillScriptPath_ResolveScriptPathRejectsSymlinkEscape(t *testing.T) {
	skill := makeBoundarySkill(t, filepath.Join(t.TempDir(), "repo"), "target-skill")
	outsidePath := filepath.Join(t.TempDir(), "outside.sh")
	mustWriteSkillScriptBoundaryFile(t, outsidePath, "echo outside\n")
	linkPath := filepath.Join(skill.Directory, "scripts", "escape.sh")
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if _, err := resolveScriptPathForTool(skill, "escape.sh"); err == nil {
		t.Fatal("resolveScriptPathForTool(escape.sh) error = nil, want symlink escape rejection")
	}
}

func makeBoundarySkill(t *testing.T, root, name string) skillcatalog.ParsedSkill {
	t.Helper()
	skillDir := filepath.Join(root, ".agents", "skills", name)
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return skillcatalog.ParsedSkill{
		Name:      name,
		Directory: skillDir,
	}
}

func mustWriteSkillScriptBoundaryFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

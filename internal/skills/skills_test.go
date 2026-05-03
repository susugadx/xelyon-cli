package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSKILL_WithResources(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, `---
name: demo-skill
description: Demonstration skill
---
# Demo
Use this for demo.
`)
	mustWriteFile(t, filepath.Join(skillDir, "scripts", "run.sh"), "#!/usr/bin/env bash\necho run\n")
	mustWriteFile(t, filepath.Join(skillDir, "references", "guide.md"), "guide")
	mustWriteFile(t, filepath.Join(skillDir, "assets", "logo.txt"), "asset")

	parsed, err := ParseSKILL(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ParseSKILL() error = %v", err)
	}
	if parsed.Name != "demo-skill" {
		t.Fatalf("Name = %q, want demo-skill", parsed.Name)
	}
	if parsed.Description != "Demonstration skill" {
		t.Fatalf("Description = %q", parsed.Description)
	}
	if !strings.Contains(parsed.Body, "# Demo") {
		t.Fatalf("Body should contain markdown content, got:\n%s", parsed.Body)
	}

	wantScripts := []string{"scripts/run.sh"}
	assertStringSliceEqual(t, parsed.Scripts, wantScripts)
	assertStringSliceEqual(t, parsed.References, []string{"references/guide.md"})
	assertStringSliceEqual(t, parsed.Assets, []string{"assets/logo.txt"})
}

func TestDiscover_FindsProjectAndHomeSkills(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()

	projectSkillDir := filepath.Join(workspace, ".agents", "skills", "project-skill")
	homeSkillDir := filepath.Join(home, ".agents", "skills", "home-skill")
	mustWriteSkill(t, projectSkillDir, validSkill("project", "project desc", "# project"))
	mustWriteSkill(t, homeSkillDir, validSkill("home", "home desc", "# home"))

	nested := filepath.Join(workspace, "pkg", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	result := Discover(DiscoverOptions{InvocationCWD: nested, HomeDir: home})
	if len(result.Skills) != 2 {
		t.Fatalf("Discover() skills = %d, want 2", len(result.Skills))
	}

	foundProject := false
	foundHome := false
	for _, skill := range result.Skills {
		switch skill.Source {
		case SourceProject:
			foundProject = true
		case SourceHome:
			foundHome = true
		}
	}
	if !foundProject || !foundHome {
		t.Fatalf("Discover() sources = %#v, want both project and home", result.Skills)
	}
}

func TestDiscover_HomeOnlySkillIsClassifiedAsHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	workspace := filepath.Join(home, "dev", "project")

	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	homeSkillDir := filepath.Join(home, ".agents", "skills", "home-only")
	mustWriteSkill(t, homeSkillDir, validSkill("home-only", "home desc", "# home"))

	result := Discover(DiscoverOptions{InvocationCWD: workspace, HomeDir: home})
	if len(result.Skills) != 1 {
		t.Fatalf("Discover() skills = %d, want 1", len(result.Skills))
	}
	if result.Skills[0].Source != SourceHome {
		t.Fatalf("Discover() source = %s, want %s", result.Skills[0].Source, SourceHome)
	}
	if len(result.Roots) != 2 {
		t.Fatalf("Discover() roots = %v, want project+home roots", result.Roots)
	}
}

func TestDiscover_HomeRootFallbackKeepsHomeSource(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	homeSkillDir := filepath.Join(home, ".agents", "skills", "only-home")
	mustWriteSkill(t, homeSkillDir, validSkill("only-home", "home desc", "# home"))

	result := Discover(DiscoverOptions{InvocationCWD: home, HomeDir: home})
	if len(result.Skills) != 1 {
		t.Fatalf("Discover() skills = %d, want 1", len(result.Skills))
	}
	if result.Skills[0].Source != SourceHome {
		t.Fatalf("Discover() source = %s, want %s", result.Skills[0].Source, SourceHome)
	}
	if len(result.Roots) != 1 {
		t.Fatalf("Discover() roots = %v, want only home root", result.Roots)
	}
}

func TestCatalog_SkipsMissingDescription(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "broken")
	mustWriteSkill(t, skillDir, `---
name: broken
---
# Broken
`)

	discover := DiscoverResult{Skills: []DiscoveredSkill{{
		Directory: skillDir,
		SkillPath: filepath.Join(skillDir, "SKILL.md"),
		Source:    SourceProject,
		RootOrder: 0,
		PathOrder: skillDir,
	}}}
	catalog := Catalog(discover)
	if len(catalog.Skills) != 0 {
		t.Fatalf("Catalog() skills = %d, want 0", len(catalog.Skills))
	}
	if len(catalog.Diagnostics) == 0 {
		t.Fatal("Catalog() diagnostics should include parse failure")
	}
	if !strings.Contains(catalog.Diagnostics[0].Message, "description") {
		t.Fatalf("diagnostic message = %q, want missing description", catalog.Diagnostics[0].Message)
	}
}

func TestCatalog_DuplicateNameDeterministic(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()

	projectSkillDir := filepath.Join(workspace, ".agents", "skills", "same-project")
	homeSkillDir := filepath.Join(home, ".agents", "skills", "same-home")
	mustWriteSkill(t, projectSkillDir, validSkill("same-name", "project wins", "# project"))
	mustWriteSkill(t, homeSkillDir, validSkill("same-name", "home loses", "# home"))

	catalog := Catalog(Discover(DiscoverOptions{InvocationCWD: workspace, HomeDir: home}))
	if len(catalog.Skills) != 1 {
		t.Fatalf("Catalog() skills = %d, want 1", len(catalog.Skills))
	}
	if !strings.Contains(catalog.Skills[0].SkillPath, "same-project") {
		t.Fatalf("chosen skill = %s, want project skill", catalog.Skills[0].SkillPath)
	}

	hasDuplicateWarning := false
	for _, diag := range catalog.Diagnostics {
		if diag.Code == "duplicate_skill_name" {
			hasDuplicateWarning = true
			break
		}
	}
	if !hasDuplicateWarning {
		t.Fatalf("expected duplicate_skill_name warning, diagnostics=%#v", catalog.Diagnostics)
	}
}

func TestActivateSkill_OutputIncludesMetadataAndBody(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "activate")
	mustWriteSkill(t, skillDir, validSkill("activate-me", "activate description", "# Body\nDo work."))
	mustWriteFile(t, filepath.Join(skillDir, "scripts", "run.sh"), "echo run")

	catalog := Catalog(DiscoverResult{Skills: []DiscoveredSkill{{
		Directory: skillDir,
		SkillPath: filepath.Join(skillDir, "SKILL.md"),
		Source:    SourceProject,
		RootOrder: 0,
		PathOrder: skillDir,
	}}})

	activated, err := Activate(catalog, "activate-me")
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	got := activated.Content
	for _, fragment := range []string{
		`"name": "activate-me"`,
		`"skill_directory":`,
		`"scripts": [`,
		`"scripts/run.sh"`,
		`"skill_md":`,
		"Do work.",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("Activate() output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "echo run") {
		t.Fatalf("Activate() should not include script file contents:\n%s", got)
	}
}

func TestResolveScriptPath_RejectsTraversalAndSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".agents", "skills", "script")
	mustWriteSkill(t, skillDir, validSkill("script-skill", "desc", "# body"))
	mustWriteFile(t, filepath.Join(skillDir, "scripts", "safe.sh"), "echo safe")

	skill := ParsedSkill{Directory: skillDir}
	if _, err := ResolveScriptPath(skill, "../outside.sh"); err == nil {
		t.Fatal("ResolveScriptPath(../outside.sh) should reject path traversal")
	}

	resolved, err := ResolveScriptPath(skill, "safe.sh")
	if err != nil {
		t.Fatalf("ResolveScriptPath(safe.sh) error = %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(resolved), "/scripts/safe.sh") {
		t.Fatalf("ResolveScriptPath(safe.sh) = %q", resolved)
	}

	outside := filepath.Join(root, "outside.sh")
	mustWriteFile(t, outside, "echo outside")
	link := filepath.Join(skillDir, "scripts", "escape.sh")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := ResolveScriptPath(skill, "escape.sh"); err == nil {
		t.Fatal("ResolveScriptPath(escape.sh) should reject symlink escape")
	}
}

func validSkill(name, description, body string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body + "\n"
}

func mustWriteSkill(t *testing.T, skillDir, content string) {
	t.Helper()
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length = %d, want %d (got=%v, want=%v)", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

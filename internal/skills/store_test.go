package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSkillCatalogStore_Load_ReusesCacheUntilFingerprintChanges(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "desc", "# body"))

	buildCalls := 0
	buildFn := func(discover DiscoverResult) SkillCatalog {
		buildCalls++
		return Catalog(discover)
	}

	store := NewSkillCatalogStoreWithDeps(defaultSkillCatalogStoreMaxEntries, Discover, buildFn, nil)
	opts := DiscoverOptions{InvocationCWD: workspace, HomeDir: home}

	first := store.Load(opts)
	if buildCalls != 1 {
		t.Fatalf("first Load() should build once, buildCalls=%d", buildCalls)
	}
	second := store.Load(opts)
	if buildCalls != 1 {
		t.Fatalf("second Load() should reuse cache, buildCalls=%d", buildCalls)
	}
	if len(first.Skills) != len(second.Skills) {
		t.Fatalf("cached catalog skill count mismatch: first=%d second=%d", len(first.Skills), len(second.Skills))
	}

	first.Skills[0].Description = "mutated"
	third := store.Load(opts)
	if third.Skills[0].Description == "mutated" {
		t.Fatal("cache should return cloned catalog, not shared mutable slice")
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(validSkill("demo", "desc updated", "# body")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	updated := store.Load(opts)
	if buildCalls != 2 {
		t.Fatalf("Load() after fingerprint change should rebuild once, buildCalls=%d", buildCalls)
	}
	if len(updated.Skills) != 1 || updated.Skills[0].Description != "desc updated" {
		t.Fatalf("updated catalog = %#v", updated.Skills)
	}
}

func TestSkillCatalogStore_Load_DetectsSkillContentChangeEvenWithSameSizeAndMTime(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "descAA", "# body"))

	buildCalls := 0
	buildFn := func(discover DiscoverResult) SkillCatalog {
		buildCalls++
		return Catalog(discover)
	}

	store := NewSkillCatalogStoreWithDeps(defaultSkillCatalogStoreMaxEntries, Discover, buildFn, nil)
	opts := DiscoverOptions{InvocationCWD: workspace, HomeDir: home}

	first := store.Load(opts)
	if len(first.Skills) != 1 {
		t.Fatalf("initial catalog skills = %d, want 1", len(first.Skills))
	}
	if first.Skills[0].Description != "descAA" {
		t.Fatalf("initial description = %q, want descAA", first.Skills[0].Description)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	beforeInfo, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	updatedBody := validSkill("demo", "descBB", "# body") // descAA と同じ長さ
	if err := os.WriteFile(skillPath, []byte(updatedBody), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Chtimes(skillPath, beforeInfo.ModTime(), beforeInfo.ModTime()); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	second := store.Load(opts)
	if buildCalls != 2 {
		t.Fatalf("content change should rebuild even with same mtime, buildCalls=%d", buildCalls)
	}
	if len(second.Skills) != 1 || second.Skills[0].Description != "descBB" {
		t.Fatalf("updated catalog = %#v", second.Skills)
	}
}

func TestSkillCatalogStore_Load_ReusesDiscoverWhenRootStateUnchanged(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "desc", "# body"))

	discoverCalls := 0
	discoverFn := func(opts DiscoverOptions) DiscoverResult {
		discoverCalls++
		return Discover(opts)
	}

	store := NewSkillCatalogStoreWithDeps(defaultSkillCatalogStoreMaxEntries, discoverFn, Catalog, nil)
	opts := DiscoverOptions{InvocationCWD: workspace, HomeDir: home}

	_ = store.Load(opts)
	_ = store.Load(opts)
	if discoverCalls != 1 {
		t.Fatalf("discover should be reused when root state unchanged, discoverCalls=%d", discoverCalls)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(validSkill("demo", "desc2", "# body")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_ = store.Load(opts)
	if discoverCalls != 1 {
		t.Fatalf("discover should still be reused for content-only update, discoverCalls=%d", discoverCalls)
	}

	time.Sleep(2 * time.Millisecond)
	newSkillDir := filepath.Join(workspace, ".agents", "skills", "new-skill")
	mustWriteSkill(t, newSkillDir, validSkill("new", "desc", "# body"))
	_ = store.Load(opts)
	if discoverCalls != 2 {
		t.Fatalf("discover should rerun when root state changes, discoverCalls=%d", discoverCalls)
	}
}

func TestSkillCatalogStore_LRUEviction(t *testing.T) {
	base := t.TempDir()
	home := t.TempDir()
	createWorkspace := func(name, skillName string) string {
		workspace := filepath.Join(base, name)
		skillDir := filepath.Join(workspace, ".agents", "skills", skillName)
		mustWriteSkill(t, skillDir, validSkill(skillName, "desc", "# body"))
		return workspace
	}

	workspaceA := createWorkspace("a", "skill-a")
	workspaceB := createWorkspace("b", "skill-b")
	workspaceC := createWorkspace("c", "skill-c")

	buildCalls := 0
	buildFn := func(discover DiscoverResult) SkillCatalog {
		buildCalls++
		return Catalog(discover)
	}

	store := NewSkillCatalogStoreWithDeps(2, Discover, buildFn, nil)
	_ = store.Load(DiscoverOptions{InvocationCWD: workspaceA, HomeDir: home}) // build 1
	_ = store.Load(DiscoverOptions{InvocationCWD: workspaceB, HomeDir: home}) // build 2
	_ = store.Load(DiscoverOptions{InvocationCWD: workspaceA, HomeDir: home}) // hit + A を最新化
	_ = store.Load(DiscoverOptions{InvocationCWD: workspaceC, HomeDir: home}) // build 3, B がevict対象

	if buildCalls != 3 {
		t.Fatalf("buildCalls after filling LRU = %d, want 3", buildCalls)
	}

	_ = store.Load(DiscoverOptions{InvocationCWD: workspaceB, HomeDir: home}) // B はevictされているので再build
	if buildCalls != 4 {
		t.Fatalf("evicted workspace should rebuild on next load, buildCalls=%d", buildCalls)
	}
}

func TestSkillCatalogStore_Clear_DropsCachedEntries(t *testing.T) {
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "desc", "# body"))

	buildCalls := 0
	buildFn := func(discover DiscoverResult) SkillCatalog {
		buildCalls++
		return Catalog(discover)
	}

	store := NewSkillCatalogStoreWithDeps(defaultSkillCatalogStoreMaxEntries, Discover, buildFn, nil)
	opts := DiscoverOptions{InvocationCWD: workspace}

	_ = store.Load(opts)
	_ = store.Load(opts)
	if buildCalls != 1 {
		t.Fatalf("Load() before Clear should hit cache, buildCalls=%d", buildCalls)
	}

	store.Clear()
	_ = store.Load(opts)
	if buildCalls != 2 {
		t.Fatalf("Load() after Clear should rebuild, buildCalls=%d", buildCalls)
	}
}

func TestBuildCatalogFingerprint_TracksResourceListing(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "desc", "# body"))

	discover := Discover(DiscoverOptions{InvocationCWD: workspace, HomeDir: home})
	first := buildCatalogFingerprint(discover)

	mustWriteFile(t, filepath.Join(skillDir, "scripts", "run.sh"), "echo run")
	secondDiscover := Discover(DiscoverOptions{InvocationCWD: workspace, HomeDir: home})
	second := buildCatalogFingerprint(secondDiscover)
	if first == second {
		t.Fatalf("fingerprint should change when resource listing changes: %s", first)
	}

	if !strings.Contains(strings.Join(secondDiscover.Roots, ","), ".agents/skills") {
		t.Fatalf("discover roots should include skills root: %#v", secondDiscover.Roots)
	}
}

func TestCatalogWithContentCache_UsesCachedSkillContent(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(workspace, ".agents", "skills", "demo")
	mustWriteSkill(t, skillDir, validSkill("demo", "desc", "# body"))
	skillPath := filepath.Join(skillDir, "SKILL.md")

	discover := Discover(DiscoverOptions{InvocationCWD: workspace, HomeDir: home})
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if err := os.Chmod(skillPath, 0o000); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(skillPath, 0o644) })

	catalog := CatalogWithContentCache(discover, map[string][]byte{
		cleanAbsPathOrFallback(skillPath): content,
	})
	if len(catalog.Diagnostics) != 0 {
		t.Fatalf("CatalogWithContentCache() diagnostics = %#v", catalog.Diagnostics)
	}
	if len(catalog.Skills) != 1 || catalog.Skills[0].Name != "demo" {
		t.Fatalf("CatalogWithContentCache() skills = %#v", catalog.Skills)
	}
}

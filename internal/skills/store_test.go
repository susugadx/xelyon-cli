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
	opts := testDiscoverOptions(workspace, home)

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
	skill, ok := findParsedSkill(updated.Skills, "demo")
	if !ok || skill.Description != "desc updated" {
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
	opts := testDiscoverOptions(workspace, home)

	first := store.Load(opts)
	skill, ok := findParsedSkill(first.Skills, "demo")
	if !ok {
		t.Fatalf("initial catalog missing demo skill: %#v", first.Skills)
	}
	if skill.Description != "descAA" {
		t.Fatalf("initial description = %q, want descAA", skill.Description)
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
	skill, ok = findParsedSkill(second.Skills, "demo")
	if !ok || skill.Description != "descBB" {
		t.Fatalf("updated catalog = %#v", second.Skills)
	}
}

func TestSkillCatalogStore_Load_InvalidatesOnXelyonMetadataAddRemoveAndContentChange(t *testing.T) {
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
	opts := testDiscoverOptions(workspace, home)

	first := store.Load(opts)
	if buildCalls != 1 {
		t.Fatalf("first Load() should build once, buildCalls=%d", buildCalls)
	}
	skill, ok := findParsedSkill(first.Skills, "demo")
	if !ok || skill.Routing != nil {
		t.Fatalf("initial skill = %#v, want no routing metadata", skill)
	}

	metadataPath := filepath.Join(skillDir, "agents", "xelyon.yaml")
	mustWriteFile(t, metadataPath, `version: 1
intents:
  - code-review
role: primary
`)
	withMetadata := store.Load(opts)
	if buildCalls != 2 {
		t.Fatalf("adding sidecar should rebuild, buildCalls=%d", buildCalls)
	}
	skill, ok = findParsedSkill(withMetadata.Skills, "demo")
	if !ok || skill.Routing == nil || len(skill.Routing.Intents) != 1 || skill.Routing.Intents[0] != "code-review" {
		t.Fatalf("with metadata skill = %#v", skill)
	}

	if err := os.WriteFile(metadataPath, []byte(`version: 1
intents:
  - bug-investigation
role: supporting
`), 0o644); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}
	updated := store.Load(opts)
	if buildCalls != 3 {
		t.Fatalf("changing sidecar should rebuild, buildCalls=%d", buildCalls)
	}
	skill, ok = findParsedSkill(updated.Skills, "demo")
	if !ok || skill.Routing == nil || len(skill.Routing.Intents) != 1 || skill.Routing.Intents[0] != "bug-investigation" {
		t.Fatalf("updated metadata skill = %#v", skill)
	}

	if err := os.Remove(metadataPath); err != nil {
		t.Fatalf("Remove(metadata) error = %v", err)
	}
	removed := store.Load(opts)
	if buildCalls != 4 {
		t.Fatalf("removing sidecar should rebuild, buildCalls=%d", buildCalls)
	}
	skill, ok = findParsedSkill(removed.Skills, "demo")
	if !ok || skill.Routing != nil {
		t.Fatalf("removed metadata skill = %#v, want no routing metadata", skill)
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
	opts := testDiscoverOptions(workspace, home)

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

func TestSkillCatalogStore_Load_IgnoresCodexSkillRoot(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	codexSystemDir := filepath.Join(home, ".codex", "skills", ".system")
	if err := os.MkdirAll(codexSystemDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	discoverCalls := 0
	discoverFn := func(opts DiscoverOptions) DiscoverResult {
		discoverCalls++
		return Discover(opts)
	}

	store := NewSkillCatalogStoreWithDeps(defaultSkillCatalogStoreMaxEntries, discoverFn, Catalog, nil)
	opts := testDiscoverOptions(workspace, home)

	first := store.Load(opts)
	if len(first.Skills) != 1 || first.Skills[0].Name != xelyonBuiltinSkillCreatorName || first.Skills[0].Source != SourceXelyon {
		t.Fatalf("initial catalog skills = %#v, want built-in skill-creator", first.Skills)
	}
	if discoverCalls != 1 {
		t.Fatalf("initial discoverCalls = %d, want 1", discoverCalls)
	}

	time.Sleep(2 * time.Millisecond)
	mustWriteSkill(t, filepath.Join(codexSystemDir, "skill-creator"), validSkill("skill-creator", "create desc", "# body"))

	second := store.Load(opts)
	if discoverCalls != 1 {
		t.Fatalf("ignored codex skill should not rerun discover, discoverCalls=%d", discoverCalls)
	}
	if len(second.Skills) != 1 || second.Skills[0].Name != xelyonBuiltinSkillCreatorName || second.Skills[0].Source != SourceXelyon {
		t.Fatalf("updated catalog skills = %#v, want built-in skill-creator", second.Skills)
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
	_ = store.Load(testDiscoverOptions(workspaceA, home)) // build 1
	_ = store.Load(testDiscoverOptions(workspaceB, home)) // build 2
	_ = store.Load(testDiscoverOptions(workspaceA, home)) // hit + A を最新化
	_ = store.Load(testDiscoverOptions(workspaceC, home)) // build 3, B がevict対象

	if buildCalls != 3 {
		t.Fatalf("buildCalls after filling LRU = %d, want 3", buildCalls)
	}

	_ = store.Load(testDiscoverOptions(workspaceB, home)) // B はevictされているので再build
	if buildCalls != 4 {
		t.Fatalf("evicted workspace should rebuild on next load, buildCalls=%d", buildCalls)
	}
}

func TestSkillCatalogStore_Clear_DropsCachedEntries(t *testing.T) {
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
	opts := testDiscoverOptions(workspace, home)

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

	discover := Discover(testDiscoverOptions(workspace, home))
	first := buildCatalogFingerprint(discover)

	mustWriteFile(t, filepath.Join(skillDir, "scripts", "run.sh"), "echo run")
	secondDiscover := Discover(testDiscoverOptions(workspace, home))
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

	discover := Discover(testDiscoverOptions(workspace, home))
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
	if _, ok := findParsedSkill(catalog.Skills, "demo"); !ok {
		t.Fatalf("CatalogWithContentCache() skills = %#v", catalog.Skills)
	}
}

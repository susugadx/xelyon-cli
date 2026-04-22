package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendBundleItemAffectedPath_UsesResolvedPathFirst(t *testing.T) {
	opts := SearchOptions{Path: t.TempDir()}
	rootPath := t.TempDir()

	item := SymbolBundleItem{
		File:         "src/ignored.go",
		ResolvedPath: filepath.Join(rootPath, "src", "resolved.go"),
	}
	paths := appendBundleItemAffectedPath(nil, item, opts, rootPath)
	if len(paths) != 1 {
		t.Fatalf("expected one path, got %v", paths)
	}
	if paths[0] != filepath.Join(rootPath, "src", "resolved.go") {
		t.Fatalf("expected resolved path to be used, got %v", paths)
	}
}

func TestAppendBundleDependencyAffectedPaths_UsesRootBase(t *testing.T) {
	root := t.TempDir()
	paths := appendBundleDependencyAffectedPaths(nil, []string{"pkg/helper.go"}, root)
	if len(paths) != 1 {
		t.Fatalf("expected one dependency path, got %v", paths)
	}
	if paths[0] != filepath.Join(root, "pkg", "helper.go") {
		t.Fatalf("expected root-based dependency path, got %v", paths)
	}
}

func TestPrimaryAffectedPathCollectorAddRef_DedupesAndSkipsEmpty(t *testing.T) {
	collector := newPrimaryAffectedPathCollector()
	collector.addRef(primaryFileRef{DisplayPath: "a.go", ResolvedPath: ""})
	collector.addRef(primaryFileRef{DisplayPath: "a.go", ResolvedPath: "/tmp/a.go"})
	collector.addRef(primaryFileRef{DisplayPath: "a.go", ResolvedPath: "/tmp/a.go"})

	paths := collector.results()
	if len(paths) != 1 {
		t.Fatalf("expected one deduped path, got %v", paths)
	}
	if paths[0] != "/tmp/a.go" {
		t.Fatalf("unexpected path: %v", paths)
	}
}

func TestBundleAffectedFileBaseCandidates_Order(t *testing.T) {
	opts := SearchOptions{
		ProjectMapRootPath: "/workspace/project",
		InvocationCWD:      "/workspace/invoke",
	}
	bases := bundleAffectedFileBaseCandidates(opts, "/workspace/bundle-root")
	if len(bases) < 4 {
		t.Fatalf("expected at least 4 bases, got %v", bases)
	}
	if bases[0] != "/workspace/bundle-root" {
		t.Fatalf("unexpected first base: %v", bases)
	}
	if bases[1] != "/workspace/project" {
		t.Fatalf("unexpected second base: %v", bases)
	}
	if bases[2] != "/workspace/invoke" {
		t.Fatalf("unexpected third base: %v", bases)
	}
}

func TestSymbolAffectedFileBaseCandidates_Order(t *testing.T) {
	opts := SearchOptions{
		ProjectMapRootPath: "/workspace/project",
		InvocationCWD:      "/workspace/invoke",
	}
	bases := symbolAffectedFileBaseCandidates(opts, "/workspace/symbol-root")
	if len(bases) < 4 {
		t.Fatalf("expected at least 4 bases, got %v", bases)
	}
	if bases[0] != "/workspace/project" {
		t.Fatalf("unexpected first base: %v", bases)
	}
	if bases[1] != "/workspace/symbol-root" {
		t.Fatalf("unexpected second base: %v", bases)
	}
	if bases[2] != "/workspace/invoke" {
		t.Fatalf("unexpected third base: %v", bases)
	}
}

func TestAffectedFileBaseCandidates_SkipsEmptyPriorities(t *testing.T) {
	opts := SearchOptions{InvocationCWD: "/workspace/invoke"}
	bases := affectedFileBaseCandidates(opts, " ", "/workspace/a", "")
	if len(bases) < 2 {
		t.Fatalf("expected at least prioritized and invocation base, got %v", bases)
	}
	if bases[0] != "/workspace/a" {
		t.Fatalf("unexpected prioritized base order: %v", bases)
	}
}

func TestInvocationCWDOrGetwd_NormalizesInvocationCWD(t *testing.T) {
	dir := t.TempDir()
	messy := filepath.Join(dir, "sub", "..")
	got := invocationCWDOrGetwd(SearchOptions{InvocationCWD: messy})
	if got != dir {
		t.Fatalf("expected normalized invocation cwd %q, got %q", dir, got)
	}
}

func TestAffectedFileBasePath_SymbolUsesNormalizedProjectRoot(t *testing.T) {
	dir := t.TempDir()
	messy := filepath.Join(dir, "sub", "..")
	got := affectedFileBasePath(SearchOptions{ProjectMapRootPath: messy}, affectedFileSourceSymbol)
	if got != dir {
		t.Fatalf("expected normalized symbol base %q, got %q", dir, got)
	}
}

func TestAffectedFileBasePath_UnknownSourceFallsBackToInvocationCWD(t *testing.T) {
	dir := t.TempDir()
	got := affectedFileBasePath(SearchOptions{InvocationCWD: dir}, affectedFileSource(999))
	if got != dir {
		t.Fatalf("expected unknown source to fallback to invocation cwd %q, got %q", dir, got)
	}
}

func TestResolveAffectedFileCandidateFromBases(t *testing.T) {
	root := t.TempDir()
	baseA := filepath.Join(root, "a")
	baseB := filepath.Join(root, "b")
	target := filepath.Join(baseB, "pkg", "main.go")
	if err := os.MkdirAll(filepath.Join(baseB, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolution := resolveAffectedFileCandidateFromBases("pkg/main.go", []string{baseA, baseB})
	if resolution.Matched != target {
		t.Fatalf("expected matched path %q, got %+v", target, resolution)
	}
	if resolution.Fallback != filepath.Join(baseA, "pkg", "main.go") {
		t.Fatalf("expected fallback to first base candidate, got %+v", resolution)
	}
}

func TestFallbackAbsoluteAffectedFilePath_ReturnsAbsoluteCleanPath(t *testing.T) {
	wd := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}

	rel := filepath.Join("tmp", "..", "target.go")
	got := fallbackAbsoluteAffectedFilePath(rel)
	want := filepath.Join(wd, "target.go")
	if got != want {
		t.Fatalf("expected absolute clean path %q, got %q", want, got)
	}
}

func TestNormalizedUniqueBasePaths_DedupesAndNormalizes(t *testing.T) {
	root := t.TempDir()
	bases := normalizedUniqueBasePaths([]string{
		filepath.Join(root, "a"),
		filepath.Join(root, "a", "."),
		" ",
		filepath.Join(root, "b"),
	})
	if len(bases) != 2 {
		t.Fatalf("expected 2 normalized bases, got %v", bases)
	}
	if bases[0] != filepath.Join(root, "a") {
		t.Fatalf("unexpected first base: %v", bases)
	}
	if bases[1] != filepath.Join(root, "b") {
		t.Fatalf("unexpected second base: %v", bases)
	}
}

func TestAffectedFileCandidatePath_ConvertsSlashSeparators(t *testing.T) {
	base := filepath.Join("tmp", "repo")
	got := affectedFileCandidatePath("pkg/main.go", base)
	want := filepath.Join(base, "pkg", "main.go")
	if got != want {
		t.Fatalf("expected candidate path %q, got %q", want, got)
	}
}

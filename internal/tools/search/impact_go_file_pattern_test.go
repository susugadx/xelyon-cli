package search

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactAllowsGoFilePattern(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.go": "package src\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newGoImpactFilePatternSearchOptions(dir, "ScopedImpactBuild", "*.go"))

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go structured impact for *.go; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/build.go" {
		t.Fatalf("definition file = %q, want src/build.go", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactScopesGoGlobEvidence(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.go":        "package app\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		"packages/app/src/use.go":          "package app\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/app/src/build_test.go":   "package app\n\nimport \"testing\"\n\nfunc TestScopedImpactBuild(t *testing.T) { _ = ScopedImpactBuild() }\n",
		"packages/other/src/build.go":      "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
		"packages/other/src/use.go":        "package other\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/other/src/build_test.go": "package other\n\nimport \"testing\"\n\nfunc TestScopedImpactBuild(t *testing.T) { _ = ScopedImpactBuild() }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newGoImpactFilePatternSearchOptions(dir, "ScopedImpactBuild", "packages/app/src/**/*.go"))

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go structured impact for scoped Go glob; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want scoped glob to isolate app definition; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want scoped Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "packages/app/src/build.go" {
		t.Fatalf("definition file = %q, want packages/app/src/build.go", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "packages/app/src/use.go") {
		t.Fatalf("scoped Go glob should keep app caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "packages/app/src/build_test.go") {
		t.Fatalf("scoped Go glob should keep app test evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/build.go")
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/use.go")
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/build_test.go")
	if strings.Contains(artifact.Rendered, "packages/other/src/") {
		t.Fatalf("scoped Go glob should exclude sibling package evidence, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactScopedGlobUsesWorkspaceRootForDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.go":   "package app\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		"packages/app/src/use.go":     "package app\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/other/src/build.go": "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
		"packages/other/src/use.go":   "package other\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	opts := newGoImpactWorkspaceSearchOptions(
		dir,
		dir,
		filepath.Join(dir, "packages", "app"),
		"ScopedImpactBuild",
		"packages/app/src/**/*.go",
	)
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want scoped Go glob to keep structured impact when path is already scoped; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want workspace-root glob to isolate app definition; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want scoped Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "packages/app/src/build.go" {
		t.Fatalf("definition file = %q, want packages/app/src/build.go", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "packages/app/src/use.go") {
		t.Fatalf("scoped Go glob should keep app caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/build.go")
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/use.go")
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactScopedGlobKeepsCWDRelativeEvidence(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.go":   "package app\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		"packages/app/src/use.go":     "package app\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/other/src/build.go": "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
		"packages/other/src/use.go":   "package other\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
	})
	appDir := filepath.Join(root, "packages", "app")
	withWorkingDirForGoImpactTest(t, appDir)

	opts := newGoImpactWorkspaceSearchOptions(root, appDir, appDir, "ScopedImpactBuild", "packages/app/src/**/*.go")
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want scoped Go glob to keep structured impact from scoped cwd; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want scoped Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/build.go" {
		t.Fatalf("definition file = %q, want src/build.go", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "src/use.go") {
		t.Fatalf("workspace scoped Go glob should keep cwd-relative in-scope caller evidence, sections: %+v\noutput:\n%s", artifact.Metadata.Bundle.Sections, artifact.Rendered)
	}
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/build.go")
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/use.go")
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactRelativeScopedGlobUsesSearchTarget(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/src/build.go":   "package src\n\nfunc ScopedImpactBuild() string { return \"pkg\" }\n",
		"pkg/src/use.go":     "package src\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"other/src/build.go": "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
		"other/src/use.go":   "package other\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
	})
	cwd := filepath.Join(root, "pkg")
	withWorkingDirForGoImpactTest(t, cwd)

	opts := newGoImpactWorkspaceSearchOptions(root, cwd, ".", "ScopedImpactBuild", "src/**/*.go")
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want relative scoped Go glob to keep structured impact; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want scoped Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/build.go" {
		t.Fatalf("definition file = %q, want src/build.go relative to invocation cwd", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "src/use.go") {
		t.Fatalf("relative scoped Go glob should keep in-scope caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	if strings.Contains(artifact.Rendered, "other/src/") {
		t.Fatalf("relative scoped Go glob should exclude sibling package evidence, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactRejectsTargetRelativeGlobForAbsoluteWorkspaceSubdir(t *testing.T) {
	root := setupMultiLangDir(t, map[string]string{
		"pkg/src/build.go": "package src\n\nfunc ScopedImpactBuild() string { return \"pkg\" }\n",
		"pkg/src/use.go":   "package src\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
	})
	pkgDir := filepath.Join(root, "pkg")
	withWorkingDirForGoImpactTest(t, pkgDir)

	opts := newGoImpactWorkspaceSearchOptions(root, pkgDir, pkgDir, "ScopedImpactBuild", "src/**/*.go")
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = true, want fallback because workspace-relative glob excludes absolute subdir target; output:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "src/build.go") || strings.Contains(artifact.Rendered, "src/use.go") {
		t.Fatalf("fallback should preserve workspace-relative file_filter and exclude pkg/src files, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactScopesDirectoryEvidence(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"packages/app/src/build.go":        "package app\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		"packages/app/src/use.go":          "package app\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/app/src/build_test.go":   "package app\n\nimport \"testing\"\n\nfunc TestScopedImpactBuild(t *testing.T) { _ = ScopedImpactBuild() }\n",
		"packages/other/src/build.go":      "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
		"packages/other/src/use.go":        "package other\n\nfunc UseScopedImpactBuild() string { return ScopedImpactBuild() }\n",
		"packages/other/src/build_test.go": "package other\n\nimport \"testing\"\n\nfunc TestScopedImpactBuild(t *testing.T) { _ = ScopedImpactBuild() }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	opts := newGoImpactFilePatternSearchOptions(dir, "ScopedImpactBuild", "")
	opts.Path = filepath.Join(dir, "packages", "app", "src")
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go structured impact for scoped directory; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want scoped directory to isolate app definition; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want scoped Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Definition.File; got != "packages/app/src/build.go" {
		t.Fatalf("definition file = %q, want packages/app/src/build.go", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "packages/app/src/use.go") {
		t.Fatalf("scoped directory should keep app caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "packages/app/src/build_test.go") {
		t.Fatalf("scoped directory should keep app test evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/use.go")
	assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "packages/other/src/build_test.go")
	if strings.Contains(artifact.Rendered, "packages/other/src/") {
		t.Fatalf("scoped directory should exclude sibling package evidence, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactPreservesRescueEndToEnd(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"target.go": "package app\n\nfunc Target() string { return \"app\" }\n",
		"use.go":    "package app\n\nfunc UseTarget() string { return Target() }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	opts := newGoImpactFilePatternSearchOptions(dir, `Target\(`, "")
	opts.FileType = "go"
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go rescue query to use structured impact; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Identity.Query; got != "Target" {
		t.Fatalf("bundle query = %q, want rescued symbol Target", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "use.go") {
		t.Fatalf("Go rescue structured impact should keep caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactPreservesRescueWithGoGlob(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"target.go": "package app\n\nfunc Target() string { return \"app\" }\n",
		"use.go":    "package app\n\nfunc UseTarget() string { return Target() }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newGoImpactFilePatternSearchOptions(dir, `Target\(`, "*.go"))

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go rescue query with glob filter to use structured impact; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if got := artifact.Metadata.Bundle.Identity.Query; got != "Target" {
		t.Fatalf("bundle query = %q, want rescued symbol Target", got)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "use.go") {
		t.Fatalf("Go rescue structured impact should keep caller evidence, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactScopesEvidenceBeforeBudgets(t *testing.T) {
	files := map[string]string{
		"zapp/build.go": "package zapp\n\nfunc Target() string { return \"app\" }\n",
		"zapp/use.go":   "package zapp\n\nfunc UseTarget() string { return Target() }\n",
	}
	for i := 0; i < 12; i++ {
		files[fmt.Sprintf("a%02d/use.go", i)] = fmt.Sprintf("package a%02d\n\nfunc UseTarget%d() string { return Target() }\n", i, i)
	}
	dir := setupMultiLangDir(t, files)
	withWorkingDirForGoImpactTest(t, dir)

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newGoImpactFilePatternSearchOptions(dir, "Target", "zapp/**/*.go"))

	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want Go structured impact for scoped Go glob; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want Go structured impact bundle; output:\n%s", artifact.Rendered)
	}
	if !bundleSectionsContainFile(artifact.Metadata.Bundle, "zapp/use.go") {
		t.Fatalf("scoped Go glob should keep in-scope caller after out-of-scope references, sections: %+v", artifact.Metadata.Bundle.Sections)
	}
	for i := 0; i < 12; i++ {
		assertGoImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, fmt.Sprintf("a%02d/use.go", i))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_GoStructuredImpactPreservesNarrowGlobFallback(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.go":      "package src\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		"src/build_test.go": "package src\n\nfunc TestScopedImpactBuild() { _ = ScopedImpactBuild }\n",
	})
	withWorkingDirForGoImpactTest(t, dir)

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newGoImpactFilePatternSearchOptions(dir, "ScopedImpactBuild", "*_test.go"))

	if artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = true, want fallback for narrow Go glob; output:\n%s", artifact.Rendered)
	}
	if !strings.Contains(artifact.Rendered, "src/build_test.go") {
		t.Fatalf("fallback should preserve *_test.go scope, got:\n%s", artifact.Rendered)
	}
	if strings.Contains(artifact.Rendered, "src/build.go") {
		t.Fatalf("fallback should not include non-test Go definition for *_test.go, got:\n%s", artifact.Rendered)
	}
}

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadProjectInstructionBundle_BasenameLoadsRootToCwdScopedChain(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	cwd := filepath.Join(root, "app", "pkg")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "root claude\n")
	writeFile(t, filepath.Join(root, "app", "AGENTS.md"), "app\n")
	writeFile(t, filepath.Join(root, "app", "CLAUDE.md"), "app claude\n")
	writeFile(t, filepath.Join(root, "app", "pkg", "AGENTS.md"), "pkg\n")
	writeFile(t, filepath.Join(root, "app", "pkg", "CLAUDE.md"), "pkg claude\n")
	initGitRepo(t, root)

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.Project.Files = []string{"AGENTS.md", "CLAUDE.md"}

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, cwd)
	gotLabels := instructionFileLabels(bundle.ProjectGuidance)
	wantLabels := []string{
		"AGENTS.md",
		"CLAUDE.md",
		"app/AGENTS.md",
		"app/CLAUDE.md",
		"app/pkg/AGENTS.md",
		"app/pkg/CLAUDE.md",
	}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("labels = %#v, want %#v", gotLabels, wantLabels)
	}
	gotScopes := instructionFileRepositoryScopes(bundle.ProjectGuidance)
	wantScopes := []string{".", ".", "app", "app", "app/pkg", "app/pkg"}
	if !reflect.DeepEqual(gotScopes, wantScopes) {
		t.Fatalf("repository scopes = %#v, want %#v", gotScopes, wantScopes)
	}
}

func TestLoadProjectInstructionBundle_InputPathAddsScopedChain(t *testing.T) {
	root := t.TempDir()
	inputFile := filepath.Join(root, "internal", "agent", "compress.go")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "root claude\n")
	writeFile(t, filepath.Join(root, "internal", "AGENTS.md"), "internal\n")
	writeFile(t, filepath.Join(root, "internal", "CLAUDE.md"), "internal claude\n")
	writeFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "agent\n")
	writeFile(t, filepath.Join(root, "internal", "agent", "CLAUDE.md"), "agent claude\n")
	writeFile(t, inputFile, "package agent\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.Project.Files = []string{"AGENTS.md", "CLAUDE.md"}

	bundle, err := LoadProjectInstructionBundleForDirWithInputPaths(cfg, root, []string{"internal/agent/compress.go"})
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDirWithInputPaths() error = %v", err)
	}
	gotLabels := instructionFileLabels(bundle.ProjectGuidance)
	wantLabels := []string{
		"AGENTS.md",
		"CLAUDE.md",
		"internal/AGENTS.md",
		"internal/CLAUDE.md",
		"internal/agent/AGENTS.md",
		"internal/agent/CLAUDE.md",
	}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("labels = %#v, want %#v", gotLabels, wantLabels)
	}
}

func TestLoadProjectInstructionBundle_SlashEntryStaysRootRelativeOnly(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	cwd := filepath.Join(root, "src")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(root, "src", "AGENTS.md"), "src\n")
	writeFile(t, filepath.Join(root, "docs", "AGENTS.md"), "docs\n")
	initGitRepo(t, root)

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.Project.Files = []string{"AGENTS.md", "docs/AGENTS.md"}

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, cwd)
	gotLabels := instructionFileLabels(bundle.ProjectGuidance)
	wantLabels := []string{"AGENTS.md", "docs/AGENTS.md", "src/AGENTS.md"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("labels = %#v, want %#v", gotLabels, wantLabels)
	}
	if bundle.ProjectGuidance[1].RepositoryScope != "." {
		t.Fatalf("explicit slash entry scope = %q, want .", bundle.ProjectGuidance[1].RepositoryScope)
	}
}

func TestComputeProjectInstructionBundleFingerprintWithInputPathsTracksSelectedNestedGuidance(t *testing.T) {
	root := t.TempDir()
	nestedPath := filepath.Join(root, "internal", "AGENTS.md")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, nestedPath, "NESTED_V1\n")
	writeFile(t, filepath.Join(root, "internal", "agent.go"), "package internal\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	baseBefore, baseAfter := fingerprintBeforeAfter(t, cfg, root, func() {
		overwriteFileAndBumpMTime(t, nestedPath, "NESTED_V2\n")
	})
	assertFingerprintStable(t, baseBefore, baseAfter, "base fingerprint should not change when unselected nested guidance changes")

	selectedBefore := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"internal/agent.go"}, nil)
	overwriteFileAndBumpMTime(t, nestedPath, "NESTED_V3\n")
	selectedAfter := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"internal/agent.go"}, nil)
	assertFingerprintChanged(t, selectedBefore, selectedAfter, "input-selected fingerprint should change when nested guidance changes")
}

func TestComputeProjectInstructionBundleFingerprintWithInputPathsChangesBySelectedSubtree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(root, "internal", "AGENTS.md"), "internal\n")
	writeFile(t, filepath.Join(root, "pkg", "AGENTS.md"), "pkg\n")
	writeFile(t, filepath.Join(root, "internal", "agent.go"), "package internal\n")
	writeFile(t, filepath.Join(root, "pkg", "pkg.go"), "package pkg\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	internalKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"internal/agent.go"}, nil)
	pkgKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"pkg/pkg.go"}, nil)
	assertFingerprintChanged(t, internalKey, pkgKey, "fingerprint should change when input selects a different guidance subtree")
}

func TestComputeProjectInstructionBundleFingerprintWithInputPathsIgnoresInvalidInputPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	baseKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, nil, nil)
	invalidKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"../outside.go", "missing.go"}, nil)
	assertFingerprintStable(t, baseKey, invalidKey, "invalid or missing input paths should not affect the guidance fingerprint")
}

func TestComputeProjectInstructionBundleFingerprintWithInputPathsUsesDirectoryScope(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(root, "internal", "a.go"), "package internal\n")
	writeFile(t, filepath.Join(root, "internal", "b.go"), "package internal\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	first := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"internal/a.go"}, nil)
	second := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"internal/b.go"}, nil)
	assertFingerprintStable(t, first, second, "different files in the same directory scope should reuse the same guidance fingerprint")
}

func TestLoadProjectInstructionBundle_InputPathSymlinkDirectoryOutsideRootSkipped(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root\n")
	writeFile(t, filepath.Join(outside, "AGENTS.md"), "outside\n")
	writeFile(t, filepath.Join(outside, "outside.go"), "package outside\n")
	createSymlinkOrSkip(t, outside, filepath.Join(root, "link"))

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	baseKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, nil, nil)
	symlinkKey := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, []string{"link/outside.go"}, nil)
	assertFingerprintStable(t, baseKey, symlinkKey, "outside-root symlink input paths should not affect selected guidance fingerprint")

	bundle, err := LoadProjectInstructionBundleForDirWithInputPaths(cfg, root, []string{"link/outside.go"})
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDirWithInputPaths() error = %v", err)
	}
	gotLabels := instructionFileLabels(bundle.ProjectGuidance)
	wantLabels := []string{"AGENTS.md"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("labels = %#v, want %#v", gotLabels, wantLabels)
	}
}

func instructionFileLabels(files []InstructionFile) []string {
	labels := make([]string, 0, len(files))
	for _, file := range files {
		labels = append(labels, file.Label)
	}
	return labels
}

func instructionFileRepositoryScopes(files []InstructionFile) []string {
	scopes := make([]string, 0, len(files))
	for _, file := range files {
		scopes = append(scopes, file.RepositoryScope)
	}
	return scopes
}

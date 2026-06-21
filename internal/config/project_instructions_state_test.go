package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func bumpFileMTime(t *testing.T, path string) {
	t.Helper()
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
}

func overwriteFileAndBumpMTime(t *testing.T, path, content string) {
	t.Helper()
	writeFile(t, path, content)
	bumpFileMTime(t, path)
}

func fingerprintBeforeAfter(t *testing.T, cfg *Config, cwd string, mutate func()) (string, string) {
	t.Helper()
	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, cwd, nil)
	if mutate != nil {
		mutate()
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, cwd, nil)
	return before, after
}

func assertFingerprintChanged(t *testing.T, before, after, reason string) {
	t.Helper()
	if before == after {
		t.Fatalf("%s\nbefore=%q\nafter=%q", reason, before, after)
	}
}

func assertFingerprintStable(t *testing.T, before, after, reason string) {
	t.Helper()
	if before != after {
		t.Fatalf("%s\nbefore=%q\nafter=%q", reason, before, after)
	}
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsTracksImportedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeFile(t, policyPath, "POLICY_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		overwriteFileAndBumpMTime(t, policyPath, "POLICY_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change when imported guidance changes with expand_imports=true")
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsDisabledIgnoresImportedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeFile(t, policyPath, "POLICY_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = false

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		overwriteFileAndBumpMTime(t, policyPath, "POLICY_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should remain stable when expand_imports=false")
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsHandlesCyclicImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeFile(t, policyPath, "POLICY_V1\n@AGENTS.md\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	overwriteFileAndBumpMTime(t, policyPath, "POLICY_V2\n@AGENTS.md\n")
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	assertFingerprintChanged(t, before, after, "fingerprint should change when cyclic imported file changes")
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsIgnoresOutsideRootImport(t *testing.T) {
	workRoot := t.TempDir()
	root := filepath.Join(workRoot, "repo")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@../outside.md\nafter\n")
	outsidePath := filepath.Join(workRoot, "outside.md")
	writeFile(t, outsidePath, "OUTSIDE_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		overwriteFileAndBumpMTime(t, outsidePath, "OUTSIDE_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should not track outside-root imports")
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsTracksMissingImportState(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@missing.md\nafter\n")
	missingPath := filepath.Join(root, "missing.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		overwriteFileAndBumpMTime(t, missingPath, "NOW_EXISTS\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change when missing import file becomes present")
}

func TestComputeProjectInstructionBundleFingerprintForDir_ProjectModeOffIgnoresProjectGuidanceChanges(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeOff
	cfg.AgentInstructions.Project.IncludeGitignored = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should remain stable in project.mode=off")
}

func TestComputeProjectInstructionBundleFingerprintForDir_FallbackWithXelyonTracksProjectGuidanceChanges(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "xelyon.yaml"), "context: test\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeFallback
	cfg.AgentInstructions.Project.IncludeGitignored = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change in fallback mode when xelyon.yaml exists")
}

func TestComputeProjectInstructionBundleFingerprintForDir_FallbackWithoutXelyonTracksTrackedProjectGuidance(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeFallback

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change in fallback mode without xelyon.yaml when tracked guidance changes")
}

func TestComputeProjectInstructionBundleFingerprintForDir_AlwaysModeTracksTrackedProjectGuidance(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeAlways

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change in project.mode=always when tracked guidance changes")
}

func TestComputeProjectInstructionBundleFingerprintForDir_IncludeGitignoredFalseIgnoresUntrackedGuidanceChanges(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")
	initGitRepo(t, root)

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeAlways
	cfg.AgentInstructions.Project.IncludeGitignored = false

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should remain stable when include_gitignored=false and guidance is untracked")
}

func TestComputeProjectInstructionBundleFingerprintForDir_IncludeGitignoredFalseIgnoresGitignoredGuidanceChanges(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "AGENTS.md\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")
	initGitRepo(t, root)
	runGit(t, root, "add", ".gitignore")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeAlways
	cfg.AgentInstructions.Project.IncludeGitignored = false

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should remain stable when include_gitignored=false and guidance is gitignored")
}

func TestComputeProjectInstructionBundleFingerprintForDir_IncludeGitignoredTrueTracksUntrackedGuidanceChanges(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V1\n")
	initGitRepo(t, root)

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeAlways
	cfg.AgentInstructions.Project.IncludeGitignored = true

	before, after := fingerprintBeforeAfter(t, cfg, root, func() {
		writeFile(t, filepath.Join(root, "AGENTS.md"), "AGENTS_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change when include_gitignored=true and guidance is untracked")
}

func TestComputeProjectInstructionBundleFingerprintForDir_GlobalDisabledIgnoresGlobalGuidanceChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	globalPath := filepath.Join(home, ".xelyon", "AGENTS.md")
	writeFile(t, globalPath, "GLOBAL_V1\n")
	workspace := t.TempDir()

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeOff
	cfg.AgentInstructions.Global.Enabled = false

	before, after := fingerprintBeforeAfter(t, cfg, workspace, func() {
		writeFile(t, globalPath, "GLOBAL_V2\n")
	})
	assertFingerprintStable(t, before, after, "fingerprint should remain stable when global guidance is disabled")
}

func TestComputeProjectInstructionBundleFingerprintForDir_GlobalEnabledTracksGlobalGuidanceChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	globalPath := filepath.Join(home, ".xelyon", "AGENTS.md")
	writeFile(t, globalPath, "GLOBAL_V1\n")
	workspace := t.TempDir()

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = AgentInstructionProjectModeOff
	cfg.AgentInstructions.Global.Enabled = true

	before, after := fingerprintBeforeAfter(t, cfg, workspace, func() {
		writeFile(t, globalPath, "GLOBAL_V2\n")
	})
	assertFingerprintChanged(t, before, after, "fingerprint should change when global guidance is enabled")
}

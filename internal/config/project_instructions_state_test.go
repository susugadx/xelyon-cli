package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsTracksImportedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeFile(t, policyPath, "POLICY_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	writeFile(t, policyPath, "POLICY_V2\n")
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(policyPath, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before == after {
		t.Fatalf("fingerprint should change when imported guidance changes with expand_imports=true\nbefore=%q\nafter=%q", before, after)
	}
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsDisabledIgnoresImportedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeFile(t, policyPath, "POLICY_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = false

	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	writeFile(t, policyPath, "POLICY_V2\n")
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(policyPath, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before != after {
		t.Fatalf("fingerprint should remain stable when expand_imports=false\nbefore=%q\nafter=%q", before, after)
	}
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
	writeFile(t, policyPath, "POLICY_V2\n@AGENTS.md\n")
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(policyPath, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before == after {
		t.Fatalf("fingerprint should change when cyclic imported file changes\nbefore=%q\nafter=%q", before, after)
	}
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

	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	writeFile(t, outsidePath, "OUTSIDE_V2\n")
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(outsidePath, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before != after {
		t.Fatalf("fingerprint should not track outside-root imports\nbefore=%q\nafter=%q", before, after)
	}
}

func TestComputeProjectInstructionBundleFingerprintForDir_ExpandImportsTracksMissingImportState(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@missing.md\nafter\n")
	missingPath := filepath.Join(root, "missing.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	before := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	writeFile(t, missingPath, "NOW_EXISTS\n")
	nextTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(missingPath, nextTime, nextTime); err != nil {
		t.Fatal(err)
	}
	after := ComputeProjectInstructionBundleFingerprintForDir(cfg, root, nil)
	if before == after {
		t.Fatalf("fingerprint should change when missing import file becomes present\nbefore=%q\nafter=%q", before, after)
	}
}

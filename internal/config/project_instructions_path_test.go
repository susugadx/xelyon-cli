package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectInstructionBundle_ProjectGuidancePathTraversalSkipped(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(base, "outside.md"), "# outside\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.Project.Files = []string{"../outside.md"}

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
	if len(bundle.WarningMessages()) == 0 {
		t.Fatal("expected warning for invalid path traversal guidance")
	}
}

func TestLoadProjectInstructionBundle_ProjectGuidanceAbsolutePathSkipped(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outsidePath := filepath.Join(base, "outside.md")
	writeFile(t, outsidePath, "# outside\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.Project.Files = []string{outsidePath}

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
	if len(bundle.WarningMessages()) == 0 {
		t.Fatal("expected warning for absolute project guidance path")
	}
}

func TestLoadProjectInstructionBundle_TrackedSymlinkOutsideRootSkipped(t *testing.T) {
	requireGit(t)

	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outsidePath := filepath.Join(base, "secret.md")
	writeFile(t, outsidePath, "# secret outside repo\n")

	initGitRepo(t, root)
	createSymlinkOrSkip(t, outsidePath, filepath.Join(root, "AGENTS.md"))
	runGit(t, root, "add", "AGENTS.md")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

func TestLoadProjectInstructionBundle_TrackedSymlinkInsideRootLoaded(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "policy.md"), "# in-repo guidance\n")
	initGitRepo(t, root)
	createSymlinkOrSkip(t, "policy.md", filepath.Join(root, "AGENTS.md"))
	runGit(t, root, "add", "AGENTS.md", "policy.md")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if !strings.Contains(bundle.ProjectGuidance[0].Content, "in-repo guidance") {
		t.Fatalf("unexpected guidance content: %q", bundle.ProjectGuidance[0].Content)
	}
}

func TestLoadProjectInstructionBundle_SymlinkWorkspaceLoadsProjectGuidance(t *testing.T) {
	base := t.TempDir()
	realRoot := filepath.Join(base, "real-root")
	symlinkRoot := filepath.Join(base, "workspace-link")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(realRoot, "AGENTS.md"), "# symlink workspace guidance\n")
	createSymlinkOrSkip(t, realRoot, symlinkRoot)

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), symlinkRoot)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Label != "AGENTS.md" {
		t.Fatalf("Label = %q, want AGENTS.md", bundle.ProjectGuidance[0].Label)
	}
	if !strings.Contains(bundle.ProjectGuidance[0].Content, "symlink workspace guidance") {
		t.Fatalf("unexpected guidance content: %q", bundle.ProjectGuidance[0].Content)
	}
}

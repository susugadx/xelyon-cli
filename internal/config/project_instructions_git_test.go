package config

import (
	"path/filepath"
	"testing"
)

func TestLoadProjectInstructionBundle_TrackedAGENTSWithoutXelyon(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# tracked agents\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if bundle.ProjectConfig != nil {
		t.Fatal("ProjectConfig should be nil")
	}
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	g := bundle.ProjectGuidance[0]
	if g.Label != "AGENTS.md" {
		t.Fatalf("Label = %q, want AGENTS.md", g.Label)
	}
	if g.Strength != InstructionStrengthProjectGuidance {
		t.Fatalf("Strength = %q, want %q", g.Strength, InstructionStrengthProjectGuidance)
	}
	if !g.GitTracked {
		t.Fatal("expected GitTracked=true")
	}
}

func TestLoadProjectInstructionBundle_TrackedCLAUDEWithoutXelyon(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "# tracked claude\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "CLAUDE.md")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Label != "CLAUDE.md" {
		t.Fatalf("Label = %q, want CLAUDE.md", bundle.ProjectGuidance[0].Label)
	}
}

func TestLoadProjectInstructionBundle_UntrackedGuidanceSkippedByDefault(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# untracked\n")
	initGitRepo(t, root)

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

func TestLoadProjectInstructionBundle_GitignoredGuidanceSkippedByDefault(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "AGENTS.md\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# ignored\n")
	initGitRepo(t, root)
	runGit(t, root, "add", ".gitignore")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

func TestLoadProjectInstructionBundle_IncludeGitignoredLoadsUntrackedGuidance(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# untracked\n")
	initGitRepo(t, root)

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].GitTracked {
		t.Fatal("untracked guidance should not be marked GitTracked")
	}
}

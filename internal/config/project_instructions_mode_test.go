package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig_AGENTSOnlyStillNil(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc != nil {
		t.Fatalf("LoadProjectConfig() = %+v, want nil", pc)
	}
}

func TestLoadProjectInstructionBundle_FallbackModeSkipsGuidanceWhenXelyonExists(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "xelyon.yaml"), "context: test\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# tracked agents\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md", "xelyon.yaml")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = "fallback"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if bundle.ProjectConfig == nil {
		t.Fatal("expected ProjectConfig to be loaded")
	}
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

func TestLoadProjectInstructionBundle_AlwaysModeLoadsAdvisoryWithXelyon(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "xelyon.yaml"), "context: test\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# tracked agents\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md", "xelyon.yaml")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = "always"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Strength != InstructionStrengthAdvisory {
		t.Fatalf("Strength = %q, want %q", bundle.ProjectGuidance[0].Strength, InstructionStrengthAdvisory)
	}
}

func TestLoadProjectInstructionBundle_OffModeSkipsProjectGuidance(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# guidance\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = "off"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

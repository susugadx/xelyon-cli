package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadProjectInstructionBundle_FallbackModeLoadsGuidanceWhenXelyonExists(t *testing.T) {
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
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Strength != InstructionStrengthProjectGuidance {
		t.Fatalf("Strength = %q, want %q", bundle.ProjectGuidance[0].Strength, InstructionStrengthProjectGuidance)
	}
	assertProjectModeFallbackDeprecationWarning(t, bundle)
}

func TestLoadProjectInstructionBundle_AlwaysModeKeepsProjectGuidanceWithLegacyXelyon(t *testing.T) {
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
	if bundle.ProjectGuidance[0].Strength != InstructionStrengthProjectGuidance {
		t.Fatalf("Strength = %q, want %q", bundle.ProjectGuidance[0].Strength, InstructionStrengthProjectGuidance)
	}
	for _, warning := range bundle.WarningMessages() {
		if strings.Contains(warning, "project.mode=fallback is deprecated") {
			t.Fatalf("always mode should not emit fallback deprecation warning: %#v", bundle.WarningMessages())
		}
	}
}

func TestLoadProjectInstructionBundle_AlwaysModeKeepsProjectGuidanceWithConfigOnlyXelyon(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "xelyon.yaml"), "final_checks:\n  commands:\n    - make test\n")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# tracked agents\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md", "xelyon.yaml")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = "always"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Strength != InstructionStrengthProjectGuidance {
		t.Fatalf("Strength = %q, want %q", bundle.ProjectGuidance[0].Strength, InstructionStrengthProjectGuidance)
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

func assertProjectModeFallbackDeprecationWarning(t *testing.T, bundle *ProjectInstructionBundle) {
	t.Helper()
	for _, warning := range bundle.WarningMessages() {
		if strings.Contains(warning, "project.mode=fallback is deprecated") {
			return
		}
	}
	t.Fatalf("expected fallback deprecation warning, got %#v", bundle.WarningMessages())
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectInstructionBundle_GlobalDisabledSkipsGlobalGuidance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".xelyon", "AGENTS.md"), "# global\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Global.Enabled = false
	cfg.AgentInstructions.Project.Mode = "off"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, t.TempDir())
	if len(bundle.GlobalGuidance) != 0 {
		t.Fatalf("GlobalGuidance len = %d, want 0", len(bundle.GlobalGuidance))
	}
}

func TestLoadProjectInstructionBundle_GlobalEnabledLoadsAdvisory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".xelyon", "AGENTS.md"), "# global\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Global.Enabled = true
	cfg.AgentInstructions.Project.Mode = "off"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, t.TempDir())
	if len(bundle.GlobalGuidance) != 1 {
		t.Fatalf("GlobalGuidance len = %d, want 1", len(bundle.GlobalGuidance))
	}
	if bundle.GlobalGuidance[0].Strength != InstructionStrengthAdvisory {
		t.Fatalf("Strength = %q, want %q", bundle.GlobalGuidance[0].Strength, InstructionStrengthAdvisory)
	}
}

func TestLoadProjectInstructionBundle_LocalFilesSkippedByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# guidance\n")
	writeFile(t, filepath.Join(root, "AGENTS.local.md"), "# local\n")
	writeFile(t, filepath.Join(root, "CLAUDE.local.md"), "# local\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Files = []string{"AGENTS.local.md", "CLAUDE.local.md", "AGENTS.md"}
	cfg.AgentInstructions.Project.IncludeGitignored = true

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Label != "AGENTS.md" {
		t.Fatalf("Label = %q, want AGENTS.md", bundle.ProjectGuidance[0].Label)
	}
}

func TestLoadProjectInstructionBundle_MaxFileBytesTruncatesContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("a", 80))

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.MaxFileBytes = 20
	cfg.AgentInstructions.MaxTotalBytes = 200

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	g := bundle.ProjectGuidance[0]
	if !g.Truncated {
		t.Fatal("expected truncated=true")
	}
	if !strings.Contains(g.Content, "agent_instructions.max_file_bytes") {
		t.Fatalf("missing max_file_bytes truncation note: %q", g.Content)
	}
}

func TestLoadProjectInstructionBundle_MaxTotalBytesTruncatesAcrossFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("a", 30))
	writeFile(t, filepath.Join(root, "CLAUDE.md"), strings.Repeat("b", 30))

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.MaxFileBytes = 40
	cfg.AgentInstructions.MaxTotalBytes = 50

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 2 {
		t.Fatalf("ProjectGuidance len = %d, want 2", len(bundle.ProjectGuidance))
	}
	if !bundle.ProjectGuidance[1].Truncated {
		t.Fatal("second guidance should be truncated by total limit")
	}
	if !strings.Contains(bundle.ProjectGuidance[1].Content, "agent_instructions.max_total_bytes") {
		t.Fatalf("missing max_total_bytes truncation note: %q", bundle.ProjectGuidance[1].Content)
	}
}

func TestLoadProjectInstructionBundle_ExpandImportsDisabledKeepsDirectiveLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	writeFile(t, filepath.Join(root, "policy.md"), "POLICY_FROM_IMPORT\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = false

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	content := bundle.ProjectGuidance[0].Content
	if !strings.Contains(content, "@policy.md") {
		t.Fatalf("directive line should remain when expand_imports=false: %q", content)
	}
}

func TestLoadProjectInstructionBundle_ExpandImportsEnabledInlinesDirectiveFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	writeFile(t, filepath.Join(root, "policy.md"), "POLICY_FROM_IMPORT\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	content := bundle.ProjectGuidance[0].Content
	if strings.Contains(content, "@policy.md") {
		t.Fatalf("directive line should be expanded when expand_imports=true: %q", content)
	}
	if !strings.Contains(content, "POLICY_FROM_IMPORT") {
		t.Fatalf("expanded import content missing: %q", content)
	}
}

func TestLoadProjectInstructionBundle_ExpandImportsSkipsUntrackedImportWhenGitignoredNotAllowed(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	writeFile(t, filepath.Join(root, "policy.md"), "UNTRACKED_POLICY\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.ExpandImports = true
	cfg.AgentInstructions.Project.IncludeGitignored = false

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	content := bundle.ProjectGuidance[0].Content
	if !strings.Contains(content, "@policy.md") {
		t.Fatalf("directive line should remain when untracked import is not allowed: %q", content)
	}
	for _, warning := range bundle.WarningMessages() {
		if strings.Contains(warning, "untracked/gitignored guidance") {
			t.Fatalf("untracked/gitignored skip should not emit warning: %#v", bundle.WarningMessages())
		}
	}
}

func TestLoadProjectInstructionBundle_ExpandImportsLoadsUntrackedImportWhenAllowed(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	writeFile(t, filepath.Join(root, "policy.md"), "UNTRACKED_POLICY\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	cfg := DefaultConfig()
	cfg.AgentInstructions.ExpandImports = true
	cfg.AgentInstructions.Project.IncludeGitignored = true

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	content := bundle.ProjectGuidance[0].Content
	if strings.Contains(content, "@policy.md") {
		t.Fatalf("directive line should be replaced when untracked import is allowed: %q", content)
	}
	if !strings.Contains(content, "UNTRACKED_POLICY") {
		t.Fatalf("expanded untracked import content missing: %q", content)
	}
}

func TestLoadProjectInstructionBundle_ReadErrorStillEmitsWarning(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, root)
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
	found := false
	for _, warning := range bundle.WarningMessages() {
		if strings.Contains(warning, "read error") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected read error warning, got: %#v", bundle.WarningMessages())
	}
}

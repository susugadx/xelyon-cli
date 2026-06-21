package config

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadProjectInstructionBundle_EmptyGlobalGuidanceReportsStatusOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".xelyon", "AGENTS.md"), "")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Global.Enabled = true
	cfg.AgentInstructions.Project.Mode = "off"

	bundle := loadProjectInstructionBundleForDirOrFatal(t, cfg, t.TempDir())
	if len(bundle.GlobalGuidance) != 0 {
		t.Fatalf("GlobalGuidance len = %d, want 0", len(bundle.GlobalGuidance))
	}
	if len(bundle.GlobalGuidanceStatus) != 1 {
		t.Fatalf("GlobalGuidanceStatus len = %d, want 1", len(bundle.GlobalGuidanceStatus))
	}
	status := bundle.GlobalGuidanceStatus[0]
	if status.Label != "~/.xelyon/AGENTS.md" || status.Status != InstructionFileStatusEmpty {
		t.Fatalf("GlobalGuidanceStatus[0] = %#v, want ~/.xelyon/AGENTS.md empty", status)
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
	cfg.AgentInstructions.Project.Files = []string{"AGENTS.md", "CLAUDE.md"}
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

func TestLoadProjectInstructionBundle_MaxTotalBytesPrioritizesNearestInputScope(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("r", 80))
	writeFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "NEAREST\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.MaxFileBytes = 200
	cfg.AgentInstructions.MaxTotalBytes = len("NEAREST\n")

	bundle, err := LoadProjectInstructionBundleForDirWithInputPaths(cfg, root, []string{"internal/agent/new_feature.go"})
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDirWithInputPaths() error = %v", err)
	}
	gotLabels := instructionFileLabels(bundle.ProjectGuidance)
	wantLabels := []string{"internal/agent/AGENTS.md"}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Fatalf("labels = %#v, want %#v", gotLabels, wantLabels)
	}
	if got := bundle.ProjectGuidance[0].Content; got != "NEAREST\n" {
		t.Fatalf("nearest guidance content = %q, want NEAREST", got)
	}
}

func TestComputeProjectInstructionBundleFingerprintWithInputPathsUsesNearestBudgetPriority(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "AGENTS.md")
	nearestPath := filepath.Join(root, "internal", "agent", "AGENTS.md")
	writeFile(t, rootPath, strings.Repeat("r", 80))
	writeFile(t, nearestPath, "NEAREST_V1\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.MaxFileBytes = 200
	cfg.AgentInstructions.MaxTotalBytes = len("NEAREST_V1\n")
	inputPaths := []string{"internal/agent/new_feature.go"}

	before := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, inputPaths, nil)
	overwriteFileAndBumpMTime(t, rootPath, strings.Repeat("R", 80))
	afterRootChange := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, inputPaths, nil)
	assertFingerprintStable(t, before, afterRootChange, "root guidance excluded by nearest-first total budget should not affect fingerprint")

	overwriteFileAndBumpMTime(t, nearestPath, "NEAREST_V2\n")
	afterNearestChange := ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(cfg, root, inputPaths, nil)
	assertFingerprintChanged(t, afterRootChange, afterNearestChange, "selected nearest guidance should affect fingerprint")
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

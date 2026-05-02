package config

import (
	"os"
	"os/exec"
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

func TestLoadProjectInstructionBundle_TrackedAGENTSWithoutXelyon(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# tracked agents\n")
	initGitRepo(t, root)
	runGit(t, root, "add", "AGENTS.md")

	bundle, err := LoadProjectInstructionBundleForDir(DefaultConfig(), root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(DefaultConfig(), root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].Label != "CLAUDE.md" {
		t.Fatalf("Label = %q, want CLAUDE.md", bundle.ProjectGuidance[0].Label)
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
	if len(bundle.ProjectGuidance) != 0 {
		t.Fatalf("ProjectGuidance len = %d, want 0", len(bundle.ProjectGuidance))
	}
}

func TestLoadProjectInstructionBundle_UntrackedGuidanceSkippedByDefault(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# untracked\n")
	initGitRepo(t, root)

	bundle, err := LoadProjectInstructionBundleForDir(DefaultConfig(), root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(DefaultConfig(), root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
	if len(bundle.ProjectGuidance) != 1 {
		t.Fatalf("ProjectGuidance len = %d, want 1", len(bundle.ProjectGuidance))
	}
	if bundle.ProjectGuidance[0].GitTracked {
		t.Fatal("untracked guidance should not be marked GitTracked")
	}
}

func TestLoadProjectInstructionBundle_GlobalDisabledSkipsGlobalGuidance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".xelyon", "AGENTS.md"), "# global\n")

	cfg := DefaultConfig()
	cfg.AgentInstructions.Global.Enabled = false
	cfg.AgentInstructions.Project.Mode = "off"

	bundle, err := LoadProjectInstructionBundleForDir(cfg, t.TempDir())
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, t.TempDir())
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

	bundle, err := LoadProjectInstructionBundleForDir(cfg, root)
	if err != nil {
		t.Fatalf("LoadProjectInstructionBundleForDir() error = %v", err)
	}
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

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

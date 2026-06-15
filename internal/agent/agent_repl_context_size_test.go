package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildContextSizeBlock_UsesProjectInstructionsLabel(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# guidance\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	agent := &Agent{Runtime: runtime, SystemPrompt: "base prompt"}

	block := buildContextSizeBlock(agent)
	if !strings.Contains(block, "Project instructions:") {
		t.Fatalf("context size block should include Project instructions label:\n%s", block)
	}
	if strings.Contains(block, "xelyon.yaml:") {
		t.Fatalf("legacy xelyon.yaml label should not appear:\n%s", block)
	}
}

func TestBuildContextSizeBlock_ShowsEmptyGlobalGuidance(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".xelyon"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".xelyon", "AGENTS.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.AgentInstructions.Project.Mode = config.AgentInstructionProjectModeOff
	runtime := NewAgentRuntimeWithConfig(cfg)
	agent := &Agent{Runtime: runtime, SystemPrompt: "base prompt"}

	block := buildContextSizeBlock(agent)
	if !strings.Contains(block, "Project instructions: ~0 (global: ~/.xelyon/AGENTS.md (empty))") {
		t.Fatalf("context size block should show empty global guidance:\n%s", block)
	}
}

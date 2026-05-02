package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestAgent_ProjectInstructionBundleCache_ReloadsWhenImportedGuidanceChanges(t *testing.T) {
	root := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "before\n@policy.md\nafter\n")
	policyPath := filepath.Join(root, "policy.md")
	writeTestFile(t, policyPath, "POLICY_V1\n")
	_ = os.Chdir(root)

	cfg := config.DefaultConfig()
	cfg.AgentInstructions.Project.IncludeGitignored = true
	cfg.AgentInstructions.ExpandImports = true

	agent := &Agent{Runtime: NewAgentRuntimeWithConfig(cfg)}
	first := agent.loadProjectInstructionBundleCached(true)
	second := agent.loadProjectInstructionBundleCached(false)
	if first == nil || second == nil {
		t.Fatal("expected non-nil bundles")
	}
	if first != second {
		t.Fatal("expected cached bundle pointer before import change")
	}

	writeTestFile(t, policyPath, "POLICY_V2\n")
	nextTime := time.Now().Add(2 * time.Second)
	touchTestFile(t, policyPath, nextTime)

	third := agent.loadProjectInstructionBundleCached(false)
	if third == nil {
		t.Fatal("expected non-nil bundle after imported guidance change")
	}
	if third == second {
		t.Fatal("expected cache reload when imported guidance file changes")
	}
}

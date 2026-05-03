package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectPromptRefreshDecision_InstructionBundleChangeTriggersRefresh(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	guidancePath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(guidancePath, []byte("# guidance v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.loadProjectInstructionBundleCached(true)

	if err := os.WriteFile(guidancePath, []byte("# guidance v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	decision := agent.promptManager().ProjectPromptRefreshDecision("実装方針を整理して")
	if !decision.NeedRefresh {
		t.Fatal("expected guidance change to trigger refresh")
	}
	if decision.Reason != refreshReasonInstructionChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonInstructionChanged)
	}
}

func TestProjectPromptRefreshDecision_InstructionBundleChangeTriggersRefreshWhenProjectMapDisabled(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	guidancePath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(guidancePath, []byte("# guidance v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.cfg().ProjectMap.Enabled = false
	agent.loadProjectInstructionBundleCached(true)

	if err := os.WriteFile(guidancePath, []byte("# guidance v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	decision := agent.promptManager().ProjectPromptRefreshDecision("実装方針を整理して")
	if !decision.NeedRefresh {
		t.Fatal("expected guidance change to trigger refresh even when project map is disabled")
	}
	if decision.Reason != refreshReasonInstructionChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonInstructionChanged)
	}
}
